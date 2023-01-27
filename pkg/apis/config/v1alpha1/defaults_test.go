// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package v1alpha1_test

import (
	. "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("Defaults", func() {
	Context("Issuer restriction", func() {
		DescribeTable("#SetDefaults_Config", func(config *Configuration, matcher gomegatypes.GomegaMatcher) {
			SetDefaults_Configuration(config)
			Expect(config.RestrictIssuer).To(matcher)
		},
			Entry("should set restriction to true if nil", &Configuration{}, PointTo(BeTrue())),
			Entry("should remain true", &Configuration{RestrictIssuer: pointer.Bool(true)}, PointTo(BeTrue())),
			Entry("should remain false", &Configuration{RestrictIssuer: pointer.Bool(false)}, PointTo(BeFalse())),
		)
	})
})
