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

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"

	"github.com/gardener/gardener/pkg/utils"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

	allErrs = append(allErrs, validateACME(&config.ACME, field.NewPath("acme"))...)

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
		if len(servers) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("precheckNameservers"), *acme.PrecheckNameservers, "must contain at least one DNS server IP"))
		}
		for i, server := range servers {
			if net.ParseIP(server) == nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("precheckNameservers"), *acme.PrecheckNameservers, fmt.Sprintf("invalid IP for %d. DNS server", i+1)))
			}
		}
	}

	if acme.CACertificates != nil {
		s := strings.TrimSpace(*acme.CACertificates)
		if !strings.HasPrefix(s, "-----BEGIN CERTIFICATE-----") || !strings.HasSuffix(s, "-----END CERTIFICATE-----") {
			short := s
			if len(short) > 60 {
				short = s[:30] + "..." + s[len(s)-30:]
			}
			allErrs = append(allErrs, field.Invalid(fldPath.Child("caCertificates"), short, "invalid certificate(s), expected PEM format)"))
		}
	}
	return allErrs
}
