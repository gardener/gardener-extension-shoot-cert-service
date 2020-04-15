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
	"net/url"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"

	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateCertConfig validates the passed configuration instance.
func ValidateCertConfig(config *service.CertConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateIssuers(config.Issuers, field.NewPath("issuers"))...)

	allErrs = append(allErrs, validateDNSChallengeOnShoot(config.DNSChallengeOnShoot, field.NewPath("dnsChallengeOnShoot"))...)

	return allErrs
}

func validateIssuers(issuers []service.IssuerConfig, fldPath *field.Path) field.ErrorList {
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
		names.Insert(issuer.Name)
	}

	return allErrs
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
