// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
)

// ValidateConfiguration validates the passed configuration instance.
func ValidateConfiguration(config *config.Configuration) field.ErrorList {
	allErrs := field.ErrorList{}

	if config.IssuerName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("issuerName"), "field is required"))
	}

	if config.DefaultRequestsPerDayQuota != nil && *config.DefaultRequestsPerDayQuota < 1 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("defaultRequestsPerDayQuota"), *config.DefaultRequestsPerDayQuota, "must be >= 1"))
	}

	if config.ACME != nil && config.CA != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("acme"), config.ACME, "only one of ACME or CA can be specified"))
	}
	if config.ACME == nil && config.CA == nil {
		allErrs = append(allErrs, field.Required(field.NewPath("acme"), "at least one of ACME or CA must be specified"))
	}
	if config.ACME != nil {
		allErrs = append(allErrs, validateACME(config.ACME, field.NewPath("acme"))...)
	}
	if config.CA != nil {
		allErrs = append(allErrs, validateCA(config.CA, field.NewPath("ca"))...)
	}

	allErrs = append(allErrs, validatePrivateKeyDefaults(config.PrivateKeyDefaults, field.NewPath("privateKeyDefaults"))...)

	return allErrs
}

func validateACME(acme *config.ACME, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if _, err := url.ParseRequestURI(acme.Server); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("server"), acme.Server, err.Error()))
	}

	if !utils.TestEmail(acme.Email) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("email"), acme.Email, "must be a valid mail address"))
	}

	if acme.PrecheckNameservers != nil {
		servers := strings.Split(*acme.PrecheckNameservers, ",")
		if len(servers) == 1 && len(servers[0]) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("precheckNameservers"), *acme.PrecheckNameservers, "must contain at least one DNS server IP"))
		} else {
			for i, server := range servers {
				if net.ParseIP(server) == nil {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("precheckNameservers"), *acme.PrecheckNameservers, fmt.Sprintf("invalid IP for %d. DNS server", i+1)))
				}
			}
		}
	}

	if acme.CACertificates != nil {
		if err := validateCACertificates(fldPath.Child("caCertificates"), *acme.CACertificates); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return allErrs
}

func validateCA(ca *config.CA, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if _, err := validateCertificate(fldPath.Child("certificate"), []byte(strings.TrimSpace(ca.Certificate))); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateCertificateKey(fldPath.Child("certificateKey"), []byte(strings.TrimSpace(ca.CertificateKey))); err != nil {
		allErrs = append(allErrs, err)
	}

	return allErrs
}

func validateCACertificates(fldPath *field.Path, caCertificates string) *field.Error {
	data := []byte(strings.TrimSpace(caCertificates))
	for len(data) > 0 {
		var err *field.Error
		data, err = validateCertificate(fldPath, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateCertificate(fldPath *field.Path, data []byte) ([]byte, *field.Error) {
	if len(data) == 0 {
		return nil, nil
	}

	block, rest := pem.Decode(data)
	if block == nil {
		return nil, field.Invalid(fldPath, shorten(string(data)), "invalid certificate: expected PEM format")
	}
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, field.Invalid(fldPath, shorten(string(data)), "invalid certificate")
	}
	return rest, nil
}

func validateCertificateKey(fldPath *field.Path, data []byte) *field.Error {
	if len(data) == 0 {
		return nil
	}

	block, rest := pem.Decode(data)
	if block == nil {
		return field.Invalid(fldPath, shorten(string(data)), "invalid certificate private key: expected PEM format")
	}
	_, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	if err != nil {
		_, err = x509.ParseECPrivateKey(block.Bytes)
	}
	if err != nil {
		return field.Invalid(fldPath, shorten(string(data)), "invalid certificate private key")
	}
	if len(rest) > 0 {
		return field.Invalid(fldPath, shorten(string(data)), "certificate private key contains additional data")
	}
	return nil
}

func validatePrivateKeyDefaults(defaults *config.PrivateKeyDefaults, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if defaults == nil {
		return allErrs
	}

	if defaults.Algorithm != nil && *defaults.Algorithm != "RSA" && *defaults.Algorithm != "ECDSA" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("algorithm"), *defaults.Algorithm, "algorithm must either be 'RSA' or 'ECDSA'"))
	}
	if defaults.SizeRSA != nil && *defaults.SizeRSA != 2048 && *defaults.SizeRSA != 3072 && *defaults.SizeRSA != 4096 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("sizeRSA"), *defaults.SizeRSA, "size for RSA algorithm must either be '2048' or '3072' or '4096'"))
	}
	if defaults.SizeECDSA != nil && *defaults.SizeECDSA != 256 && *defaults.SizeECDSA != 384 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("sizeECDSA"), *defaults.SizeECDSA, "size for ECDSA algorithm must either be '256' or '384'"))
	}

	return allErrs
}

func shorten(s string) string {
	if len(s) > 60 {
		return s[:30] + "..." + s[len(s)-30:]
	}
	return s
}
