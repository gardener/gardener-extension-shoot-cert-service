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
	"context"
	"time"

	apisconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck/general"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	certv1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	certcontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller"
)

var (
	defaultSyncPeriod = time.Second * 30
	// DefaultAddOptions are the default DefaultAddArgs for AddToManager.
	DefaultAddOptions = healthcheck.DefaultAddArgs{
		HealthCheckConfig: apisconfig.HealthCheckConfig{SyncPeriod: metav1.Duration{Duration: defaultSyncPeriod}},
	}
)

// RegisterHealthChecks registers health checks for each extension resource
// HealthChecks are grouped by extension (e.g worker), extension.type (e.g aws) and  Health Check Type (e.g SystemComponentsHealthy)
func RegisterHealthChecks(mgr manager.Manager, opts healthcheck.DefaultAddArgs) error {
	preCheckFunc := func(_ context.Context, _ client.Client, _ client.Object, cluster *extensionscontroller.Cluster) bool {
		return cluster.Shoot.Spec.DNS != nil && cluster.Shoot.Spec.DNS.Domain != nil
	}

	return healthcheck.DefaultRegistration(
		certcontroller.Type,
		extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ExtensionResource),
		func() client.ObjectList { return &extensionsv1alpha1.ExtensionList{} },
		func() extensionsv1alpha1.Object { return &extensionsv1alpha1.Extension{} },
		mgr,
		opts,
		nil,
		[]healthcheck.ConditionTypeToHealthCheck{
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck: NewIssuerWrapperHealthChecker(
					general.CheckManagedResource(certv1alpha1.CertManagementResourceNameSeed)),
				PreCheckFunc: preCheckFunc,
			},
		},
		// TODO(shafeeqes): Remove this condition in a future release.
		sets.New[gardencorev1beta1.ConditionType](gardencorev1beta1.ShootSystemComponentsHealthy),
	)
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return RegisterHealthChecks(mgr, DefaultAddOptions)
}
