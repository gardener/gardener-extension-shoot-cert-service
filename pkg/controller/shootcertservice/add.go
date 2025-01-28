// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shootcertservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	controllerconfig "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/config"
)

const (
	// Type is the type of Extension resource.
	Type = "shoot-cert-service"
	// ControllerName is the name of the shoot cert service controller.
	ControllerName = "shoot_cert_service"
	// FinalizerSuffix is the finalizer suffix for the shoot cert service controller.
	FinalizerSuffix = "shoot-cert-service"
	// TriggerLabel is the label to trigger an extension on startup.
	TriggerLabel = "shoot-cert-service.extensions.config.gardener.cloud/trigger"
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
	ServiceConfig controllerconfig.Config
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ExtensionClass defines the extension class this extension is responsible for.
	ExtensionClass extensionsv1alpha1.ExtensionClass
}

// AddToManager adds a controller with the default Options to the given Controller Manager.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	predicates := extension.DefaultPredicates(ctx, mgr, DefaultAddOptions.IgnoreOperationAnnotation)

	// Trigger reconciliation for existing extensions in the deployment namespace on election.
	go triggerReconcileSpecialExtensionOnElection(ctx, mgr, opts.ExtensionClass)

	return extension.Add(mgr, extension.AddArgs{
		Actuator:          NewActuator(mgr, opts.ServiceConfig.Configuration, opts.ExtensionClass),
		ControllerOptions: opts.ControllerOptions,
		Name:              ControllerName,
		FinalizerSuffix:   FinalizerSuffix,
		Resync:            0,
		Predicates:        predicates,
		Type:              Type,
		ExtensionClass:    opts.ExtensionClass,
	})
}

// IsSpecialNamespace returns true if the given namespace is either the garden namespace or an extension namespace.
func IsSpecialNamespace(namespace string) bool {
	return namespace == v1beta1constants.GardenNamespace || strings.HasPrefix(namespace, "extension-")
}

func triggerReconcileSpecialExtensionOnElection(ctx context.Context, mgr manager.Manager, class extensionsv1alpha1.ExtensionClass) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-mgr.Elected():
			if err := triggerReconcileSpecialExtension(ctx, mgr.GetLogger(), mgr.GetClient(), class); err != nil {
				mgr.GetLogger().Error(err, "failed to trigger reconciliation for existing extensions in the deployment namespace")
			}
			return
		}
	}
}

// triggerReconcileSpecialExtension triggers reconciliation for existing extension for seed in the garden namespace.
func triggerReconcileSpecialExtension(ctx context.Context, log logr.Logger, cl client.Client, class extensionsv1alpha1.ExtensionClass) error {
	log.Info("Triggering reconciliation for extensions with trigger label")
	list := &extensionsv1alpha1.ExtensionList{}
	if err := cl.List(ctx, list, client.MatchingLabels{TriggerLabel: "true"}); err != nil {
		return fmt.Errorf("failed to list extensions: %w", err)
	}
	if class == "" {
		class = extensionsv1alpha1.ExtensionClassShoot
	}
	for _, ext := range list.Items {
		if ptr.Deref(ext.Spec.Class, extensionsv1alpha1.ExtensionClassShoot) == class && ext.Spec.Type == Type {
			patch := client.MergeFrom(ext.DeepCopy())
			if ext.Annotations == nil {
				ext.Annotations = map[string]string{}
			}
			ext.Annotations[v1beta1constants.GardenerOperation] = v1beta1constants.GardenerOperationReconcile
			if err := cl.Patch(ctx, &ext, patch); err != nil {
				return fmt.Errorf("failed to patch special extension %s: %w", client.ObjectKeyFromObject(&ext), err)
			}
			log.Info("Triggered reconciliation for special extension", "extension", client.ObjectKeyFromObject(&ext))
		}
	}
	return nil
}
