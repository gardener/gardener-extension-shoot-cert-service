// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"fmt"
	"slices"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

const (
	// Type is the type of Extension resource.
	Type = "shoot-cert-service"
	// ControllerName is the name of the shoot cert service controller.
	ControllerName = "shoot-cert-service"
	// dnsServiceExtensionName is the name of the shoot-dns-service Extension resource that is watched
	// to trigger reconciliation of the shoot-cert-service Extension in the same namespace.
	dnsServiceExtensionName = "shoot-dns-service"
	// useNextGenerationControllerAnnotation is the annotation on the shoot-dns-service Extension
	// indicating whether the next-generation DNS controller is in use.
	useNextGenerationControllerAnnotation = "service.dns.extensions.gardener.cloud/use-next-generation-controller"
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

// AddToManager adds a controller with the default Options to the given Controller Manager.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	predicates := extension.DefaultPredicates(ctx, mgr, DefaultAddOptions.IgnoreOperationAnnotation)

	if slices.Contains(opts.ExtensionClasses, extensionsv1alpha1.ExtensionClassGarden) {
		return fmt.Errorf("controller %q for type %q is not supported for extension class %q", ControllerName, Type, extensionsv1alpha1.ExtensionClassGarden)
	}

	extensionClasses := []extensionsv1alpha1.ExtensionClass{extensionsv1alpha1.ExtensionClassShoot}

	watchBuilder := extensionscontroller.NewWatchBuilder(func(c controller.Controller) error {
		return c.Watch(source.Kind(
			mgr.GetCache(),
			&extensionsv1alpha1.Extension{},
			handler.TypedEnqueueRequestsFromMapFunc(mapDNSServiceExtensionToCertServiceExtension()),
			&dnsServiceExtensionPredicate{},
		))
	})

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

// mapDNSServiceExtensionToCertServiceExtension maps a shoot-dns-service Extension event to a reconcile
// request for the shoot-cert-service Extension in the same namespace.
func mapDNSServiceExtensionToCertServiceExtension() func(context.Context, *extensionsv1alpha1.Extension) []reconcile.Request {
	return func(_ context.Context, ex *extensionsv1alpha1.Extension) []reconcile.Request {
		if ex == nil || ex.Name != dnsServiceExtensionName {
			return nil
		}
		return []reconcile.Request{{
			NamespacedName: client.ObjectKey{
				Name:      Type,
				Namespace: ex.Namespace,
			},
		}}
	}
}

// dnsServiceExtensionPredicate filters Extension events to only those for the shoot-dns-service
// Extension on create, and on update only when the next-generation controller annotation changes.
type dnsServiceExtensionPredicate struct{}

func (dnsServiceExtensionPredicate) Create(e event.TypedCreateEvent[*extensionsv1alpha1.Extension]) bool {
	return e.Object != nil && e.Object.Name == dnsServiceExtensionName
}

func (dnsServiceExtensionPredicate) Update(e event.TypedUpdateEvent[*extensionsv1alpha1.Extension]) bool {
	if e.ObjectNew == nil || e.ObjectNew.Name != dnsServiceExtensionName {
		return false
	}
	oldValue := ""
	if e.ObjectOld != nil {
		oldValue = e.ObjectOld.Annotations[useNextGenerationControllerAnnotation]
	}
	return oldValue != e.ObjectNew.Annotations[useNextGenerationControllerAnnotation]
}

func (dnsServiceExtensionPredicate) Delete(_ event.TypedDeleteEvent[*extensionsv1alpha1.Extension]) bool {
	return false
}

func (dnsServiceExtensionPredicate) Generic(_ event.TypedGenericEvent[*extensionsv1alpha1.Extension]) bool {
	return false
}
