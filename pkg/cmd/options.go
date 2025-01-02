// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"os"

	extensionsapisconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/controller/cmd"
	extensionshealthcheckcontroller "github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	extensionsheartbeatcontroller "github.com/gardener/gardener/extensions/pkg/controller/heartbeat"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	apisconfig "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/validation"
	controllerconfig "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/config"
	healthcheckcontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/healthcheck"
	certificatecontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
	gardencontroller "github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/garden"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/shootcertservice"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/webhook/sniconfig"
)

var (
	scheme  *runtime.Scheme
	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()
	utilruntime.Must(apisconfig.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// CertificateServiceOptions holds options related to the certificate service.
type CertificateServiceOptions struct {
	ConfigLocation string
	config         *CertificateServiceConfig
}

// AddFlags implements Flagger.AddFlags.
func (o *CertificateServiceOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to cert service configuration")
}

// Complete implements Completer.Complete.
func (o *CertificateServiceOptions) Complete() error {
	if o.ConfigLocation == "" {
		return errors.New("config location is not set")
	}
	data, err := os.ReadFile(o.ConfigLocation)
	if err != nil {
		return err
	}

	config := apisconfig.Configuration{}
	_, _, err = decoder.Decode(data, nil, &config)
	if err != nil {
		return err
	}

	if errs := validation.ValidateConfiguration(&config); len(errs) > 0 {
		return errs.ToAggregate()
	}

	o.config = &CertificateServiceConfig{
		config: config,
	}

	return nil
}

// Completed returns the decoded CertificatesServiceConfiguration instance. Only call this if `Complete` was successful.
func (o *CertificateServiceOptions) Completed() *CertificateServiceConfig {
	return o.config
}

// CertificateServiceConfig contains configuration information about the certificate service.
type CertificateServiceConfig struct {
	config apisconfig.Configuration
}

// Apply applies the CertificateServiceOptions to the passed ControllerOptions instance.
func (c *CertificateServiceConfig) Apply(config *controllerconfig.Config) {
	config.Configuration = c.config
}

// ControllerSwitches are the cmd.SwitchOptions for the provider controllers.
func ControllerSwitches() *cmd.SwitchOptions {
	return cmd.NewSwitchOptions(
		cmd.Switch(shootcertservice.ControllerName, shootcertservice.AddToManager),
		cmd.Switch(extensionshealthcheckcontroller.ControllerName, healthcheckcontroller.AddToManager),
		cmd.Switch(extensionsheartbeatcontroller.ControllerName, extensionsheartbeatcontroller.AddToManager),
		cmd.Switch(gardencontroller.ControllerName, gardencontroller.AddToManager),
		cmd.Switch(certificatecontroller.ControllerName, certificatecontroller.AddToManager),
	)
}

func (c *CertificateServiceConfig) ApplyHealthCheckConfig(config *extensionsapisconfigv1alpha1.HealthCheckConfig) {
	if c.config.HealthCheckConfig != nil {
		*config = *c.config.HealthCheckConfig
	}
}

func WebhookSwitches() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(sniconfig.HandlerName, sniconfig.AddToManager),
	)
}
