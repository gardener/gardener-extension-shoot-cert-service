// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ControllerName is the name of this controller.
const ControllerName = "certificate"

// DefaultAddOptions are the default AddOptions for AddToManager.
var DefaultAddOptions = controller.Options{
	MaxConcurrentReconciles: 1,
}

// AddToManager adds a controller with the default Options to the given Controller Manager.
func AddToManager(_ context.Context, mgr manager.Manager) error {
	return (&Reconciler{
		ControllerOptions: DefaultAddOptions,
	}).AddToManager(mgr)
}

// AddToManager adds Reconciler to the given manager.
func (r *Reconciler) AddToManager(mgr manager.Manager) error {
	var err error

	if r.RuntimeClientSet == nil {
		r.RuntimeClientSet, err = kubernetes.NewWithConfig(
			kubernetes.WithRESTConfig(mgr.GetConfig()),
			kubernetes.WithRuntimeAPIReader(mgr.GetAPIReader()),
			kubernetes.WithRuntimeClient(mgr.GetClient()),
			kubernetes.WithRuntimeCache(mgr.GetCache()),
		)
		if err != nil {
			return fmt.Errorf("failed creating runtime clientset: %w", err)
		}
	}
	if r.GardenNamespace == "" {
		r.GardenNamespace = v1beta1constants.GardenNamespace
	}

	return builder.
		ControllerManagedBy(mgr).
		Named(ControllerName).
		For(&certv1alpha1.Certificate{}, builder.WithPredicates(
			extensionspredicate.IsInGardenNamespacePredicate,
			predicate.NewPredicateFuncs(func(obj client.Object) bool {
				return obj != nil &&
					obj.GetLabels()[v1beta1constants.GardenRole] == v1beta1constants.GardenRoleControlPlaneWildcardCert &&
					obj.GetLabels()[ExtensionClassLabel] == string(extensionsv1alpha1.ExtensionClassGarden)
			}),
		)).
		WithOptions(r.ControllerOptions).
		Complete(r)
}
