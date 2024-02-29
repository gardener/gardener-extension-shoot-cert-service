// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
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
		if len(issuer.PrecheckNameservers) > 0 {
			for j, server := range issuer.PrecheckNameservers {
				if err := validateNameserver(server); err != nil {
					allErrs = append(allErrs, field.Invalid(indexFldPath.Child("precheckNameservers").Index(j), server, err.Error()))
				}
			}
		}
		names.Insert(issuer.Name)
	}

	return allErrs
}

func validateNameserver(server string) error {
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host = server
		port = "53"
	}
	if net.ParseIP(host) == nil && (len(validation.IsDNS1123Subdomain(strings.TrimSuffix(host, "."))) > 0 || len(strings.Trim(host, "0123456789.")) == 0) {
		return fmt.Errorf("'%s' is no valid IP address or domain name", host)
	}

	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("'%s' is no valid port", port)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("'%s' is no valid port number", port)
	}
	return nil
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
				if err := validateNameserver(server); err != nil {
					allErrs = append(allErrs, field.Invalid(fldPath, *precheckNameservers, fmt.Sprintf("invalid value for %d. DNS server %s: %s", i+1, server, err.Error())))
				}
			}
		}
	}
	return allErrs
}
