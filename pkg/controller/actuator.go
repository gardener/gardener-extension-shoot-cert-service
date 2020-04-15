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

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/imagevector"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/gardener/pkg/utils/secrets"
)

// ActuatorName is the name of the Certificate Service actuator.
const ActuatorName = "shoot-cert-service-actuator"

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(config config.Configuration) extension.Actuator {
	return &actuator{
		logger:        log.Log.WithName(ActuatorName),
		serviceConfig: config,
	}
}

type actuator struct {
	client  client.Client
	config  *rest.Config
	decoder runtime.Decoder

	serviceConfig config.Configuration

	logger logr.Logger
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	var certConfig *service.CertConfig
	if ex.Spec.ProviderConfig != nil {
		certConfig = &service.CertConfig{}
		if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, certConfig); err != nil {
			return fmt.Errorf("failed to decode provider config: %+v", err)
		}
		if errs := validation.ValidateCertConfig(certConfig); len(errs) > 0 {
			return errs.ToAggregate()
		}
	}

	if !controller.IsHibernated(cluster) {
		if err := a.createShootResources(ctx, certConfig, cluster, namespace); err != nil {
			return err
		}
	}

	return a.createSeedResources(ctx, certConfig, cluster, namespace)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	a.logger.Info("Component is being deleted", "component", "cert-management", "namespace", namespace)
	if err := a.deleteShootResources(ctx, namespace); err != nil {
		return err
	}

	return a.deleteSeedResources(ctx, namespace)
}

// InjectConfig injects the rest config to this actuator.
func (a *actuator) InjectConfig(config *rest.Config) error {
	a.config = config
	return nil
}

// InjectClient injects the controller runtime client into the reconciler.
func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

// InjectScheme injects the given scheme into the reconciler.
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (a *actuator) createIssuerValues(issuers ...service.IssuerConfig) ([]map[string]interface{}, error) {
	issuerVal := []map[string]interface{}{
		{
			"name": a.serviceConfig.IssuerName,
			"acme": map[string]interface{}{
				"email":      a.serviceConfig.ACME.Email,
				"server":     a.serviceConfig.ACME.Server,
				"privateKey": a.serviceConfig.ACME.PrivateKey,
			},
		},
	}

	for _, issuer := range issuers {
		if issuer.Name == a.serviceConfig.IssuerName {
			continue
		}
		issuerVal = append(issuerVal, map[string]interface{}{
			"name": issuer.Name,
			"acme": map[string]interface{}{
				"email":  issuer.Email,
				"server": issuer.Server,
			},
		})
	}

	return issuerVal, nil
}

func (a *actuator) createDNSChallengeOnShootValues(cfg *service.DNSChallengeOnShoot) (map[string]interface{}, error) {
	if cfg == nil || !cfg.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	if cfg.Namespace == "" {
		return nil, fmt.Errorf("missing DNSChallengeOnShoot namespace")
	}

	values := map[string]interface{}{
		"enabled":   true,
		"namespace": cfg.Namespace,
	}

	if cfg.DNSClass != nil {
		values["dnsClass"] = *cfg.DNSClass
	}

	return values, nil
}

func (a *actuator) createSeedResources(ctx context.Context, certConfig *service.CertConfig, cluster *controller.Cluster, namespace string) error {
	var (
		issuerConfig              []service.IssuerConfig
		dnsChallengeOnShootConfig *service.DNSChallengeOnShoot
	)
	if certConfig != nil {
		issuerConfig = certConfig.Issuers
		dnsChallengeOnShootConfig = certConfig.DNSChallengeOnShoot
	}
	issuers, err := a.createIssuerValues(issuerConfig...)
	if err != nil {
		return err
	}

	dnsChallengeOnShoot, err := a.createDNSChallengeOnShootValues(dnsChallengeOnShootConfig)
	if err != nil {
		return err
	}

	if cluster.Shoot.Spec.DNS == nil || cluster.Shoot.Spec.DNS.Domain == nil {
		a.logger.Info("no domain given for shoot %s/%s - aborting", cluster.Shoot.Name, cluster.Shoot.Namespace)
		return nil
	}

	shootKubeconfig, err := a.createKubeconfigForCertManagement(ctx, namespace)
	if err != nil {
		return err
	}

	certManagementConfig := map[string]interface{}{
		"replicaCount": controller.GetReplicas(cluster, 1),
		"defaultProvider": map[string]interface{}{
			"name":    a.serviceConfig.IssuerName,
			"domains": cluster.Shoot.Spec.DNS.Domain,
		},
		"issuers":             issuers,
		"dnsChallengeOnShoot": dnsChallengeOnShoot,
		"shootClusterSecret":  v1alpha1.CertManagementKubecfg,
		"podAnnotations": map[string]interface{}{
			"checksum/secret-kubeconfig": util.ComputeChecksum(shootKubeconfig.Data),
		},
	}

	certManagementConfig, err = chart.InjectImages(certManagementConfig, imagevector.ImageVector(), []string{v1alpha1.CertManagementImageName})
	if err != nil {
		return fmt.Errorf("failed to find image version for %s: %v", v1alpha1.CertManagementImageName, err)
	}

	renderer, err := chartrenderer.NewForConfig(a.config)
	if err != nil {
		return errors.Wrap(err, "could not create chart renderer")
	}

	a.logger.Info("Component is being applied", "component", "cert-management", "namespace", namespace)

	return a.createManagedResource(ctx, namespace, v1alpha1.CertManagementResourceNameSeed, "seed", renderer, v1alpha1.CertManagementChartNameSeed, certManagementConfig, nil)
}

func (a *actuator) createShootResources(ctx context.Context, certConfig *service.CertConfig, cluster *controller.Cluster, namespace string) error {
	var dnsChallengeOnShootConfig *service.DNSChallengeOnShoot
	if certConfig != nil {
		dnsChallengeOnShootConfig = certConfig.DNSChallengeOnShoot
	}

	dnsChallengeOnShoot, err := a.createDNSChallengeOnShootValues(dnsChallengeOnShootConfig)
	if err != nil {
		return err
	}

	values := map[string]interface{}{
		"shootUserName":       v1alpha1.CertManagementUserName,
		"dnsChallengeOnShoot": dnsChallengeOnShoot,
	}

	renderer, err := util.NewChartRendererForShoot(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return errors.Wrap(err, "could not create chart renderer")
	}

	return a.createManagedResource(ctx, namespace, v1alpha1.CertManagementResourceNameShoot, "", renderer, v1alpha1.CertManagementChartNameShoot, values, nil)
}

func (a *actuator) deleteSeedResources(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for seed", "namespace", namespace)

	secret := &corev1.Secret{}
	secret.SetName(v1alpha1.CertManagementKubecfg)
	secret.SetNamespace(namespace)
	if err := a.client.Delete(ctx, secret); client.IgnoreNotFound(err) != nil {
		return err
	}
	if err := controller.DeleteManagedResource(ctx, a.client, namespace, v1alpha1.CertManagementResourceNameSeed); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return controller.WaitUntilManagedResourceDeleted(timeoutCtx, a.client, namespace, v1alpha1.CertManagementResourceNameSeed)
}

func (a *actuator) deleteShootResources(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for shoot", "namespace", namespace)
	if err := controller.DeleteManagedResource(ctx, a.client, namespace, v1alpha1.CertManagementResourceNameShoot); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return controller.WaitUntilManagedResourceDeleted(timeoutCtx, a.client, namespace, v1alpha1.CertManagementResourceNameShoot)
}

func (a *actuator) createKubeconfigForCertManagement(ctx context.Context, namespace string) (*corev1.Secret, error) {
	certConfig := secrets.CertificateSecretConfig{
		Name:       v1alpha1.CertManagementKubecfg,
		CommonName: v1alpha1.CertManagementUserName,
	}

	return util.GetOrCreateShootKubeconfig(ctx, a.client, certConfig, namespace)
}

func (a *actuator) createManagedResource(ctx context.Context, namespace, name, class string, renderer chartrenderer.Interface, chartName string, chartValues map[string]interface{}, injectedLabels map[string]string) error {
	return controller.CreateManagedResourceFromFileChart(
		ctx, a.client, namespace, name, class,
		renderer, filepath.Join(v1alpha1.ChartsPath, chartName), chartName,
		chartValues, injectedLabels,
	)
}
