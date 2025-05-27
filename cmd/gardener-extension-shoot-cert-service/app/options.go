// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"os"

	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"

	certificateservicecmd "github.com/gardener/gardener-extension-shoot-cert-service/pkg/cmd"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

// ExtensionName is the name of the extension.
const ExtensionName = "extension-shoot-cert-service"

// Options holds configuration passed to the Certificate Service controller.
type Options struct {
	generalOptions                *controllercmd.GeneralOptions
	certOptions                   *certificateservicecmd.CertificateServiceOptions
	restOptions                   *controllercmd.RESTOptions
	managerOptions                *controllercmd.ManagerOptions
	shootControllerOptions        *controllercmd.ControllerOptions
	controlPlaneControllerOptions *controllercmd.ControllerOptions
	healthOptions                 *controllercmd.ControllerOptions
	heartbeatOptions              *heartbeatcmd.Options
	controllerSwitches            *controllercmd.SwitchOptions
	reconcileOptions              *controllercmd.ReconcilerOptions
	optionAggregator              controllercmd.OptionAggregator
}

// NewOptions creates a new Options instance.
func NewOptions() *Options {
	options := &Options{
		generalOptions: &controllercmd.GeneralOptions{},
		certOptions:    &certificateservicecmd.CertificateServiceOptions{},
		restOptions:    &controllercmd.RESTOptions{},
		managerOptions: &controllercmd.ManagerOptions{
			// These are default values.
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(ExtensionName),
			LeaderElectionNamespace: os.Getenv(shared.EnvLeaderElectionNamespace),
		},
		shootControllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		controlPlaneControllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 2,
		},
		healthOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		heartbeatOptions: &heartbeatcmd.Options{
			// This is a default value.
			ExtensionName:        ExtensionName,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv(shared.EnvLeaderElectionNamespace),
		},
		controllerSwitches: certificateservicecmd.ControllerSwitches(),
		reconcileOptions:   &controllercmd.ReconcilerOptions{},
	}

	options.optionAggregator = controllercmd.NewOptionAggregator(
		options.generalOptions,
		options.restOptions,
		options.managerOptions,
		options.shootControllerOptions,
		controllercmd.PrefixOption("controlplane-", options.controlPlaneControllerOptions),
		options.certOptions,
		controllercmd.PrefixOption("healthcheck-", options.healthOptions),
		controllercmd.PrefixOption("heartbeat-", options.heartbeatOptions),
		options.controllerSwitches,
		options.reconcileOptions,
	)

	return options
}
