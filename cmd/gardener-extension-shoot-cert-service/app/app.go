// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	"github.com/gardener/gardener/extensions/pkg/util"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	serviceinstall "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/install"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/healthcheck"
	certificatecontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
	gardencontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/garden"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/shootcertservice"
)

// NewServiceControllerCommand creates a new command that is used to start the Certificate Service controller.
func NewServiceControllerCommand() *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:           "shoot-cert-service-controller-manager",
		Short:         "Shoot Cert Service Controller manages components which provide certificate services.",
		SilenceErrors: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := options.optionAggregator.Complete(); err != nil {
				return fmt.Errorf("error completing options: %s", err)
			}

			if err := options.heartbeatOptions.Validate(); err != nil {
				return err
			}
			cmd.SilenceUsage = true
			return options.run(cmd.Context())
		},
	}

	options.optionAggregator.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) run(ctx context.Context) error {
	// TODO: Make these flags configurable via command line parameters or component config file.
	util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfigv1alpha1.ClientConnectionConfiguration{
		QPS:   100.0,
		Burst: 130,
	}, o.restOptions.Completed().Config)

	mgrOpts := o.managerOptions.Completed().Options()

	mgrOpts.Client = client.Options{
		Cache: &client.CacheOptions{
			DisableFor: []client.Object{
				&corev1.Secret{},    // applied for ManagedResources
				&corev1.ConfigMap{}, // applied for monitoring config
			},
		},
	}

	mgr, err := manager.New(o.restOptions.Completed().Config, mgrOpts)
	if err != nil {
		return fmt.Errorf("could not instantiate controller-manager: %s", err)
	}

	if err := extensionscontroller.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %s", err)
	}

	if err := serviceinstall.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %s", err)
	}

	if err := certv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %s", err)
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("could not update manager scheme: %s", err)
	}

	ctrlConfig := o.certOptions.Completed()
	ctrlConfig.ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
	ctrlConfig.Apply(&shootcertservice.DefaultAddOptions.ServiceConfig)
	o.controllerOptions.Completed().Apply(&shootcertservice.DefaultAddOptions.ControllerOptions)
	o.healthOptions.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
	o.reconcileOptions.Completed().Apply(&shootcertservice.DefaultAddOptions.IgnoreOperationAnnotation, &shootcertservice.DefaultAddOptions.ExtensionClass)
	o.heartbeatOptions.Completed().Apply(&heartbeat.DefaultAddOptions)
	o.gardenControllerOptions.Completed().Apply(&gardencontroller.DefaultAddOptions)
	o.certificateControllerOptions.Completed().Apply(&certificatecontroller.DefaultAddOptions)

	if err := o.controllerSwitches.Completed().AddToManager(ctx, mgr); err != nil {
		return fmt.Errorf("could not add controllers to manager: %s", err)
	}

	if config := o.webhookOptions.Completed(); !config.Switch.Disabled {
		if _, err := config.AddToManager(ctx, mgr, mgr); err != nil {
			return fmt.Errorf("could not add webhooks to manager: %s", err)
		}
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("error running manager: %s", err)
	}

	return nil
}
