// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

const (
	// Type is the type of Extension resource.
	Type = "shoot-cert-service"
	// ControllerName is the name of the shoot cert service controller.
	ControllerName = "shoot-cert-service"
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
	// ExtensionClass defines the main extension class this extension is responsible for.
	ExtensionClass extensionsv1alpha1.ExtensionClass
}

// AddToManager adds a controller with the default Options to the given Controller Manager.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	predicates := extension.DefaultPredicates(ctx, mgr, DefaultAddOptions.IgnoreOperationAnnotation)

	if opts.ExtensionClass == extensionsv1alpha1.ExtensionClassGarden {
		return fmt.Errorf("controller %q for type %q is not supported for extension class %q", ControllerName, Type, opts.ExtensionClass)
	}

	extensionClasses := []extensionsv1alpha1.ExtensionClass{extensionsv1alpha1.ExtensionClassShoot}

	return extension.Add(mgr, extension.AddArgs{
		Actuator:          NewActuator(mgr, opts.ServiceConfig, extensionClasses),
		ControllerOptions: opts.ControllerOptions,
		Name:              ControllerName,
		FinalizerSuffix:   shared.FinalizerSuffix,
		Resync:            0,
		Predicates:        predicates,
		Type:              Type,
		ExtensionClasses:  extensionClasses,
	})
}
