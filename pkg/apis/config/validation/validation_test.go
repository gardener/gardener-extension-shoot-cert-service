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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/validation"
)

var _ = Describe("Validation", func() {
	validACME := config.ACME{
		Email:  "john.doe@example.com",
		Server: "https://acme-v02.api.letsencrypt.org/directory",
	}

	DescribeTable("#ValidateConfiguration",
		func(config config.Configuration, match gomegatypes.GomegaMatcher) {
			err := validation.ValidateConfiguration(&config)
			Expect(err).To(match)
		},
		Entry("Empty configuration", config.Configuration{
			IssuerName: "",
			ACME:       config.ACME{},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("issuerName"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.server"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.email"),
			})),
		)),
		Entry("Invalid ACME configuration", config.Configuration{
			IssuerName: "gardener",
			ACME: config.ACME{
				Email:  "john.doe.com",
				Server: "acme-v02.api.letsencrypt.org/directory",
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.server"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.email"),
			})),
		)),
		Entry("Invalid precheck nameservers and caCertificates", config.Configuration{
			IssuerName: "gardener",
			ACME: config.ACME{
				Email:               validACME.Email,
				Server:              validACME.Server,
				PrecheckNameservers: ptr.To("8.8.8.8,foo.com"),
				CACertificates:      ptr.To("blabla"),
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.precheckNameservers"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.caCertificates"),
			})),
		)),
		Entry("Valid precheck nameservers and caCertificates", config.Configuration{
			IssuerName: "gardener",
			ACME: config.ACME{
				Email:               validACME.Email,
				Server:              validACME.Server,
				PrecheckNameservers: ptr.To("8.8.8.8,172.11.22.253"),
				CACertificates: ptr.To(`
-----BEGIN CERTIFICATE-----
AAABBBCCCDDD
-----END CERTIFICATE-----
`),
			},
		}, BeEmpty()),
		Entry("Invalid DefaultRequestsPerDayQuota", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(0)),
			ACME:                       validACME,
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("defaultRequestsPerDayQuota"),
			})),
		)),
		Entry("Valid configuration", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(50)),
			ACME:                       validACME,
		}, BeEmpty()),
	)
})
