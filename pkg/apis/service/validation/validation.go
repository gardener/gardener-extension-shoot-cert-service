// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateCertConfig validates the passed configuration instance.
func ValidateCertConfig(config *service.CertConfig, cluster *controller.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateIssuers(cluster, config.Issuers, field.NewPath("issuers"))...)

	allErrs = append(allErrs, validateDNSChallengeOnShoot(config.DNSChallengeOnShoot, field.NewPath("dnsChallengeOnShoot"))...)

	allErrs = append(allErrs, validatePrecheckNameservers(config.PrecheckNameservers, field.NewPath("precheckNameservers"))...)

	return allErrs
}

func validateIssuers(cluster *controller.Cluster, issuers []service.IssuerConfig, fldPath *field.Path) field.ErrorList {
	var (
		allErrs = field.ErrorList{}
		names   = sets.NewString()
	)

	for i, issuer := range issuers {
		indexFldPath := fldPath.Index(i)
		if issuer.Name == "" {
			allErrs = append(allErrs, field.Invalid(indexFldPath.Child("name"), issuer.Name, "must not be empty"))
		}
		if names.Has(issuer.Name) {
			allErrs = append(allErrs, field.Duplicate(indexFldPath.Child("name"), issuer.Name))
		}
		if _, err := url.ParseRequestURI(issuer.Server); err != nil {
			allErrs = append(allErrs, field.Invalid(indexFldPath.Child("server"), issuer.Server, "must be a valid url"))
		}
		if !utils.TestEmail(issuer.Email) {
			allErrs = append(allErrs, field.Invalid(indexFldPath.Child("email"), issuer.Email, "must a valid email address"))
		}
		if issuer.PrivateKeySecretName != nil {
			detail := checkReferencedResource(cluster, *issuer.PrivateKeySecretName)
			if detail != "" {
				allErrs = append(allErrs, field.Invalid(indexFldPath.Child("privateKeySecretName"),
					*issuer.PrivateKeySecretName, detail))
			}
		}
		if issuer.ExternalAccountBinding != nil {
			if issuer.ExternalAccountBinding.KeyID == "" {
				allErrs = append(allErrs, field.Invalid(indexFldPath.Child("externalAccountBinding").Child("keyID"),
					issuer.ExternalAccountBinding.KeyID, "must not be empty"))
			}
			detail := checkReferencedResource(cluster, issuer.ExternalAccountBinding.KeySecretName)
			if detail != "" {
				allErrs = append(allErrs, field.Invalid(indexFldPath.Child("externalAccountBinding").Child("keySecretName"),
					issuer.ExternalAccountBinding.KeySecretName, detail))
			}
		}
		if issuer.SkipDNSChallengeValidation != nil && *issuer.SkipDNSChallengeValidation &&
			issuer.ExternalAccountBinding == nil {
			allErrs = append(allErrs, field.Invalid(indexFldPath.Child("skipDNSChallengeValidation"),
				*issuer.SkipDNSChallengeValidation, "is only allowed for external account binding"))
		}
		if issuer.RequestsPerDayQuota != nil && *issuer.RequestsPerDayQuota < 1 {
			allErrs = append(allErrs, field.Invalid(indexFldPath.Child("requestsPerDayQuota"), *issuer.RequestsPerDayQuota, "must be >= 1"))
		}
		names.Insert(issuer.Name)
	}

	return allErrs
}

func checkReferencedResource(cluster *controller.Cluster, refname string) string {
	if cluster.Shoot == nil {
		return "shoot spec not set"
	}
	for _, ref := range cluster.Shoot.Spec.Resources {
		if ref.Name == refname {
			if ref.ResourceRef.Kind != "Secret" {
				return "expected secret resource"
			}
			return "" // ok
		}
	}
	return "referenced resource not found"
}

func validateDNSChallengeOnShoot(dnsChallenge *service.DNSChallengeOnShoot, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if dnsChallenge != nil && dnsChallenge.Enabled {
		if dnsChallenge.Namespace == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("namespace"), "must provide namespace for writing DNS entries"))
		}
	}

	return allErrs
}

func validatePrecheckNameservers(precheckNameservers *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if precheckNameservers != nil {
		servers := strings.Split(*precheckNameservers, ",")
		if len(servers) == 1 && len(servers[0]) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath, *precheckNameservers, "must contain at least one DNS server IP"))
		} else {
			for i, server := range servers {
				if net.ParseIP(server) == nil {
					allErrs = append(allErrs, field.Invalid(fldPath, *precheckNameservers, fmt.Sprintf("invalid IP for %d. DNS server", i+1)))
				}
			}
		}
	}
	return allErrs
}
