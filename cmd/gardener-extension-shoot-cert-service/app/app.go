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

package app

import (
	"context"

	serviceinstall "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/install"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/healthcheck"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/spf13/cobra"
	componentbaseconfig "k8s.io/component-base/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewServiceControllerCommand creates a new command that is used to start the Certificate Service controller.
func NewServiceControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "shoot-cert-service-controller-manager",
		Short: "Shoot Cert Service Controller manages components which provide certificate services.",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.optionAggregator.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}
			options.run(ctx)
		},
	}

	options.optionAggregator.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) run(ctx context.Context) {
	// TODO: Make these flags configurable via command line parameters or component config file.
	util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfig.ClientConnectionConfiguration{
		QPS:   100.0,
		Burst: 130,
	}, o.restOptions.Completed().Config)

	mgr, err := manager.New(o.restOptions.Completed().Config, o.managerOptions.Completed().Options())
	if err != nil {
		controllercmd.LogErrAndExit(err, "Could not instantiate controller-manager")
	}

	if err := extensionscontroller.AddToScheme(mgr.GetScheme()); err != nil {
		controllercmd.LogErrAndExit(err, "Could not update manager scheme")
	}

	if err := serviceinstall.AddToScheme(mgr.GetScheme()); err != nil {
		controllercmd.LogErrAndExit(err, "Could not update manager scheme")
	}

	ctrlConfig := o.certOptions.Completed()
	ctrlConfig.ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckConfig)
	ctrlConfig.Apply(&controller.DefaultAddOptions.ServiceConfig)
	o.controllerOptions.Completed().Apply(&controller.DefaultAddOptions.ControllerOptions)
	o.healthOptions.Completed().Apply(&healthcheck.DefaultAddOptions.Controller)
	o.reconcileOptions.Completed().Apply(&controller.DefaultAddOptions.IgnoreOperationAnnotation)

	if err := o.controllerSwitches.Completed().AddToManager(mgr); err != nil {
		controllercmd.LogErrAndExit(err, "Could not add controllers to manager")
	}

	if err := mgr.Start(ctx.Done()); err != nil {
		controllercmd.LogErrAndExit(err, "Error running manager")
	}
}
