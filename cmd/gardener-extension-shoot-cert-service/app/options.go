// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"os"

	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscmdwebhook "github.com/gardener/gardener/extensions/pkg/webhook/cmd"

	certificateservicecmd "github.com/gardener/gardener-extension-shoot-cert-service/pkg/cmd"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension"
)

// ExtensionName is the name of the extension.
const ExtensionName = "extension-shoot-cert-service"

// Options holds configuration passed to the Certificate Service controller.
type Options struct {
	generalOptions               *controllercmd.GeneralOptions
	certOptions                  *certificateservicecmd.CertificateServiceOptions
	restOptions                  *controllercmd.RESTOptions
	managerOptions               *controllercmd.ManagerOptions
	controllerOptions            *controllercmd.ControllerOptions
	healthOptions                *controllercmd.ControllerOptions
	heartbeatOptions             *heartbeatcmd.Options
	gardenControllerOptions      *controllercmd.ControllerOptions
	certificateControllerOptions *controllercmd.ControllerOptions
	controllerSwitches           *controllercmd.SwitchOptions
	reconcileOptions             *controllercmd.ReconcilerOptions
	optionAggregator             controllercmd.OptionAggregator
	webhookOptions               *extensionscmdwebhook.AddToManagerOptions
}

// NewOptions creates a new Options instance.
func NewOptions() *Options {
	mode, url := extensionswebhook.ModeService, os.Getenv("WEBHOOK_URL")
	if v := os.Getenv("WEBHOOK_MODE"); v != "" {
		mode = v
	}

	options := &Options{
		generalOptions: &controllercmd.GeneralOptions{},
		certOptions:    &certificateservicecmd.CertificateServiceOptions{},
		restOptions:    &controllercmd.RESTOptions{},
		managerOptions: &controllercmd.ManagerOptions{
			// These are default values.
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(ExtensionName),
			LeaderElectionNamespace: os.Getenv(extension.EnvLeaderElectionNamespace),

			// These are default values.
			WebhookServerPort: 10250,
		},
		controllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		healthOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		gardenControllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 1,
		},
		certificateControllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 1,
		},
		heartbeatOptions: &heartbeatcmd.Options{
			// This is a default value.
			ExtensionName:        ExtensionName,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv(extension.EnvLeaderElectionNamespace),
		},
		controllerSwitches: certificateservicecmd.ControllerSwitches(),
		reconcileOptions:   &controllercmd.ReconcilerOptions{},
		webhookOptions: extensionscmdwebhook.NewAddToManagerOptions(
			"shoot-cert-service",
			"",
			nil,
			&extensionscmdwebhook.ServerOptions{
				Mode:        mode,
				URL:         url,
				ServicePort: 443,
				Namespace:   os.Getenv(extension.EnvLeaderElectionNamespace),
			},
			certificateservicecmd.WebhookSwitches()),
	}

	options.optionAggregator = controllercmd.NewOptionAggregator(
		options.generalOptions,
		options.restOptions,
		options.managerOptions,
		options.controllerOptions,
		options.certOptions,
		controllercmd.PrefixOption("healthcheck-", options.healthOptions),
		controllercmd.PrefixOption("heartbeat-", options.heartbeatOptions),
		controllercmd.PrefixOption("garden-", options.gardenControllerOptions),
		controllercmd.PrefixOption("certificate-", options.certificateControllerOptions),
		options.controllerSwitches,
		options.reconcileOptions,
		options.webhookOptions,
	)

	return options
}
