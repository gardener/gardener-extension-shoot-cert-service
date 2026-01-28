// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	operatorpredicate "github.com/gardener/gardener/pkg/operator/predicate"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

const (
	// Type is the second type of Extension resource with different life cycle (before kube-apiserver)
	Type = "controlplane-cert-service"
	// ControllerName is the name of the shoot cert service controller.
	ControllerName = "controlplane-cert-service"

	// GardenRelevantDataHashAnnotation is the annotation key for the hash of garden relevant data.
	GardenRelevantDataHashAnnotation = "garden-relevant-data-hash"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the shoot cert service controller to the manager.
type AddOptions struct {
	// ControllerOptions contains options for the controller.
	ControllerOptions controller.Options
	// ServiceConfig contains configuration for the shoot cert service.
	ServiceConfig config.Configuration
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ExtensionClasses defines the main extension classes this extension is responsible for.
	ExtensionClasses []extensionsv1alpha1.ExtensionClass
}

// AddToManager adds a second controller with the default Options to the given Controller Manager.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}

// AddToManagerWithOptions adds a second controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	var (
		predicates       = extension.DefaultPredicates(ctx, mgr, DefaultAddOptions.IgnoreOperationAnnotation)
		extensionClasses = []extensionsv1alpha1.ExtensionClass{extensionsv1alpha1.ExtensionClassSeed}
		watchBuilder     extensionscontroller.WatchBuilder
	)

	if slices.Contains(opts.ExtensionClasses, extensionsv1alpha1.ExtensionClassGarden) {
		extensionClasses = []extensionsv1alpha1.ExtensionClass{extensionsv1alpha1.ExtensionClassGarden}
		watchBuilder = extensionscontroller.NewWatchBuilder(func(c controller.Controller) error {
			return c.Watch(source.Kind(
				mgr.GetCache(),
				&operatorv1alpha1.Garden{},
				handler.TypedEnqueueRequestsFromMapFunc(mapGardenToExtension(mgr, mgr.GetLogger().WithName("mapGardenToExtension"))),
				&toTypedPredicate{predicate: operatorpredicate.GardenCreatedOrReconciledSuccessfully()},
			))
		})
	}

	return extension.Add(mgr, extension.AddArgs{
		Actuator:          NewActuator(mgr, opts.ServiceConfig, extensionClasses),
		ControllerOptions: opts.ControllerOptions,
		Name:              ControllerName,
		FinalizerSuffix:   shared.FinalizerSuffix,
		Resync:            0,
		Predicates:        predicates,
		Type:              Type,
		ExtensionClasses:  extensionClasses,
		WatchBuilder:      watchBuilder,
	})
}

type toTypedPredicate struct {
	predicate predicate.Predicate
}

func (p *toTypedPredicate) Create(e event.TypedCreateEvent[*operatorv1alpha1.Garden]) bool {
	return p.predicate.Create(event.CreateEvent{Object: e.Object})
}

func (p *toTypedPredicate) Update(e event.TypedUpdateEvent[*operatorv1alpha1.Garden]) bool {
	return p.predicate.Update(event.UpdateEvent{ObjectOld: e.ObjectOld, ObjectNew: e.ObjectNew})
}

func (p *toTypedPredicate) Delete(e event.TypedDeleteEvent[*operatorv1alpha1.Garden]) bool {
	return p.predicate.Delete(event.DeleteEvent{Object: e.Object})
}

func (p *toTypedPredicate) Generic(e event.TypedGenericEvent[*operatorv1alpha1.Garden]) bool {
	return p.predicate.Generic(event.GenericEvent{Object: e.Object})
}

func mapGardenToExtension(mgr manager.Manager, log logr.Logger) func(context.Context, *operatorv1alpha1.Garden) []reconcile.Request {
	c := mgr.GetClient()
	decoder := shared.NewCertConfigDecoder(mgr)
	return func(ctx context.Context, garden *operatorv1alpha1.Garden) []reconcile.Request {
		extList := &extensionsv1alpha1.ExtensionList{}
		if err := c.List(ctx, extList, client.InNamespace(constants.GardenNamespace)); err != nil {
			log.Error(err, "Failed to list extensions")
			return nil
		}

		var requests []reconcile.Request
		for _, ex := range extList.Items {
			if ex.Spec.Type == Type &&
				extensionsv1alpha1helper.GetExtensionClassOrDefault(ex.Spec.Class) == extensionsv1alpha1.ExtensionClassGarden {
				certConfig, err := decoder.DecodeAndValidateProviderConfig(&ex, nil)
				if err != nil {
					log.Error(err, "Failed to decode extension config")
					return nil
				}
				if ptr.Deref(certConfig.GenerateControlPlaneCertificate, false) {
					hash := calcGardenRelevantDataHash(garden)
					if ex.Annotations[GardenRelevantDataHashAnnotation] != hash {
						log.Info("Garden relevant data hash has changed, requeueing extension", "hash", hash)
						requests = append(requests, reconcile.Request{
							NamespacedName: client.ObjectKey{
								Name:      ex.Name,
								Namespace: constants.GardenNamespace,
							},
						})
					}
				}
			}
		}
		return requests
	}
}

func calcGardenRelevantDataHash(garden *operatorv1alpha1.Garden) string {
	hash := sha256.New()
	for _, domain := range garden.Spec.VirtualCluster.DNS.Domains {
		if _, err := hash.Write([]byte(domain.Name)); err != nil {
			return ""
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return ""
		}
	}

	for _, domain := range garden.Spec.RuntimeCluster.Ingress.Domains {
		if _, err := hash.Write([]byte(domain.Name)); err != nil {
			return ""
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return ""
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}
