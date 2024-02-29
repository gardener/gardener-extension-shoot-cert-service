// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/utils/ptr"

	. "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/v1alpha1"
)

var _ = Describe("Defaults", func() {
	Context("Issuer restriction", func() {
		DescribeTable("#SetDefaults_Config", func(config *Configuration, matcher gomegatypes.GomegaMatcher) {
			SetDefaults_Configuration(config)
			Expect(config.RestrictIssuer).To(matcher)
		},
			Entry("should set restriction to true if nil", &Configuration{}, PointTo(BeTrue())),
			Entry("should remain true", &Configuration{RestrictIssuer: ptr.To(true)}, PointTo(BeTrue())),
			Entry("should remain false", &Configuration{RestrictIssuer: ptr.To(false)}, PointTo(BeFalse())),
		)
	})
})
