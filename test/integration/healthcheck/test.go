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

/**
	Overview
		- Tests the health checks for the shoot-cert-service extension.

	Prerequisites
		- A Shoot exists.

	Test-case:
		1) Extension CRD
			1.1) HealthCondition Type: ShootControlPlaneHealthy
				-  update the ManagedResource 'extension-shoot-cert-service-seed' and verify the health check conditions in the Extension CRD status.

 **/

package healthcheck

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/test/framework"
	"github.com/gardener/gardener/test/testmachinery/extensions/healthcheck"
	"github.com/onsi/ginkgo/v2"

	certv1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
)

const (
	timeout = 5 * time.Minute
)

var _ = ginkgo.Describe("Extension-shoot-cert-service integration test: health checks", func() {
	f := framework.NewShootFramework(nil)

	ginkgo.Context("Extension", func() {
		ginkgo.Context("Condition type: ShootControlPlaneHealthy", func() {
			f.Serial().Release().CIt(fmt.Sprintf("Extension CRD should contain unhealthy condition due to ManagedResource '%s' is unhealthy", certv1alpha1.CertManagementResourceNameSeed), func(ctx context.Context) {
				err := healthcheck.ExtensionHealthCheckWithManagedResource(ctx, timeout, f, "shoot-cert-service", certv1alpha1.CertManagementResourceNameSeed, gardencorev1beta1.ShootControlPlaneHealthy)
				framework.ExpectNoError(err)
			}, timeout)
		})
	})
})
