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

package validation_test

import (
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Validation", func() {
	DescribeTable("#ValidateCertConfiga",
		func(config service.CertConfig, match gomegatypes.GomegaMatcher) {
			err := validation.ValidateCertConfig(&config)
			Expect(err).To(match)
		},
		Entry("No issuers", service.CertConfig{}, BeEmpty()),
		Entry("Invalid issuer", service.CertConfig{
			Issuers: []service.IssuerConfig{
				{
					Name:   "",
					Server: "",
					Email:  "",
				},
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("issuers[0].name"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("issuers[0].server"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("issuers[0].email"),
			})),
		)),
		Entry("Duplicate issuer", service.CertConfig{
			Issuers: []service.IssuerConfig{
				{
					Name:   "issuer",
					Server: "https://acme-v02.api.letsencrypt.org/directory",
					Email:  "john@example.com",
				},
				{
					Name:   "issuer",
					Server: "https://acme-v02.api.acme.org",
					Email:  "john.doe@example.com",
				},
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("issuers[1].name"),
			})),
		)),
		Entry("Valid configuration", service.CertConfig{
			Issuers: []service.IssuerConfig{
				{
					Name:   "issuer",
					Server: "https://acme-v02.api.letsencrypt.org/directory",
					Email:  "john@example.com",
				},
			},
		}, BeEmpty()),
		Entry("DNSChallengeOnShoot", service.CertConfig{
			DNSChallengeOnShoot: &service.DNSChallengeOnShoot{
				Enabled:   true,
				Namespace: "",
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("dnsChallengeOnShoot.namespace"),
			})),
		)),
		Entry("Valid DNSChallengeOnShoot", service.CertConfig{
			DNSChallengeOnShoot: &service.DNSChallengeOnShoot{
				Enabled:   true,
				Namespace: "kube-system",
			},
		}, BeEmpty()),
	)
})
