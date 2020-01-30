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

package healthcheck

import (
	"time"

	certv1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	certcontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller"

	"github.com/gardener/gardener-extensions/pkg/controller/healthcheck"
	healthcheckconfig "github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config"
	"github.com/gardener/gardener-extensions/pkg/controller/healthcheck/general"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	defaultSyncPeriod = time.Second * 30
	// DefaultAddOptions are the default DefaultAddArgs for AddToManager.
	DefaultAddOptions = healthcheck.DefaultAddArgs{
		HealthCheckConfig: healthcheckconfig.HealthCheckConfig{SyncPeriod: metav1.Duration{Duration: defaultSyncPeriod}},
	}
)

// RegisterHealthChecks registers health checks for each extension resource
// HealthChecks are grouped by extension (e.g worker), extension.type (e.g aws) and  Health Check Type (e.g SystemComponentsHealthy)
func RegisterHealthChecks(mgr manager.Manager, opts healthcheck.DefaultAddArgs) error {
	return healthcheck.DefaultRegistration(
		certcontroller.Type,
		extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ExtensionResource),
		func() runtime.Object { return &extensionsv1alpha1.Extension{} },
		mgr,
		opts,
		nil,
		map[healthcheck.HealthCheck]string{
			general.CheckManagedResource(certv1alpha1.CertManagementResourceNameSeed):  string(gardencorev1beta1.ShootControlPlaneHealthy),
			general.CheckManagedResource(certv1alpha1.CertManagementResourceNameShoot): string(gardencorev1beta1.ShootSystemComponentsHealthy),
		})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return RegisterHealthChecks(mgr, DefaultAddOptions)
}
