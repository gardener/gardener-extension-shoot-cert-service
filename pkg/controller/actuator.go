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
	"strings"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/gardener/gardener/pkg/utils/chart"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/charts"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/imagevector"
)

// ActuatorName is the name of the Certificate Service actuator.
const ActuatorName = "shoot-cert-service-actuator"

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.Configuration) extension.Actuator {
	return &actuator{
		client:        mgr.GetClient(),
		config:        mgr.GetConfig(),
		decoder:       serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
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
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	certConfig := &service.CertConfig{}
	if ex.Spec.ProviderConfig != nil {
		if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, certConfig); err != nil {
			return fmt.Errorf("failed to decode provider config: %+v", err)
		}
		if errs := validation.ValidateCertConfig(certConfig, cluster); len(errs) > 0 {
			return errs.ToAggregate()
		}
	}

	if !controller.IsHibernated(cluster) {
		if err := a.createShootResources(ctx, certConfig, cluster, namespace); err != nil {
			return err
		}
	}

	if err := a.createSeedResources(ctx, certConfig, cluster, namespace); err != nil {
		return err
	}

	return a.updateStatus(ctx, ex, certConfig)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	a.logger.Info("Component is being deleted", "component", "cert-management", "namespace", namespace)
	if err := a.deleteShootResources(ctx, namespace); err != nil {
		return err
	}

	return a.deleteSeedResources(ctx, namespace)
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	// Keep objects for shoot managed resources so that they are not deleted from the shoot during the migration
	if err := managedresources.SetKeepObjects(ctx, a.client, ex.GetNamespace(), v1alpha1.CertManagementResourceNameShoot, true); err != nil {
		return err
	}

	return a.Delete(ctx, log, ex)
}

func (a *actuator) createIssuerValues(cluster *controller.Cluster, issuers ...service.IssuerConfig) ([]map[string]interface{}, error) {
	issuerList := []map[string]interface{}{
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

		acme := map[string]interface{}{
			"email":  issuer.Email,
			"server": issuer.Server,
		}
		issuerValues := map[string]interface{}{
			"name": issuer.Name,
			"acme": acme,
		}
		if issuer.PrivateKeySecretName != nil {
			secretName := a.lookupReferencedSecret(cluster, *issuer.PrivateKeySecretName)
			acme["privateKeySecretName"] = secretName
		}
		if issuer.ExternalAccountBinding != nil {
			secretName := a.lookupReferencedSecret(cluster, issuer.ExternalAccountBinding.KeySecretName)
			acme["externalAccountBinding"] = map[string]interface{}{
				"keyID":         issuer.ExternalAccountBinding.KeyID,
				"keySecretName": secretName,
			}
		}
		if issuer.SkipDNSChallengeValidation != nil && *issuer.SkipDNSChallengeValidation {
			acme["skipDNSChallengeValidation"] = true
		}
		if issuer.Domains != nil && len(issuer.Domains.Include)+len(issuer.Domains.Exclude) > 0 {
			selection := map[string]interface{}{}
			if issuer.Domains.Include != nil {
				selection["include"] = issuer.Domains.Include
			}
			if issuer.Domains.Exclude != nil {
				selection["exclude"] = issuer.Domains.Exclude
			}
			if len(selection) > 0 {
				acme["domains"] = selection
			}
		}
		if issuer.RequestsPerDayQuota != nil {
			issuerValues["requestsPerDayQuota"] = *issuer.RequestsPerDayQuota
		}
		if len(issuer.PrecheckNameservers) > 0 {
			issuerValues["precheckNameservers"] = issuer.PrecheckNameservers
		}
		issuerList = append(issuerList, issuerValues)
	}

	return issuerList, nil
}

func (a *actuator) lookupReferencedSecret(cluster *controller.Cluster, refname string) string {
	if cluster.Shoot != nil {
		for _, ref := range cluster.Shoot.Spec.Resources {
			if ref.Name == refname {
				if ref.ResourceRef.Kind != "Secret" {
					a.logger.Info("invalid referenced resource, expected kind Secret, not %s: %s", ref.ResourceRef.Kind, refname)
					return "invalid-kind"
				}
				return v1beta1constants.ReferencedResourcesPrefix + ref.ResourceRef.Name
			}
		}
	}
	a.logger.Info("invalid referenced resource: %s", refname)
	return "invalid"
}

func createDNSChallengeOnShootValues(cfg *service.DNSChallengeOnShoot) (map[string]interface{}, error) {
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
	issuers, err := a.createIssuerValues(cluster, certConfig.Issuers...)
	if err != nil {
		return err
	}

	dnsChallengeOnShoot, err := createDNSChallengeOnShootValues(certConfig.DNSChallengeOnShoot)
	if err != nil {
		return err
	}

	if cluster.Shoot.Spec.DNS == nil || cluster.Shoot.Spec.DNS.Domain == nil {
		a.logger.Info("no domain given for shoot %s/%s - aborting", cluster.Shoot.Name, cluster.Shoot.Namespace)
		return nil
	}

	var propagationTimeout string
	if a.serviceConfig.ACME.PropagationTimeout != nil {
		propagationTimeout = a.serviceConfig.ACME.PropagationTimeout.Duration.String()
	}

	var (
		shootIssuers         = a.createShootIssuersValues(certConfig)
		certManagementConfig = map[string]interface{}{
			"replicaCount": controller.GetReplicas(cluster, 1),
			"defaultIssuer": map[string]interface{}{
				"name":       a.serviceConfig.IssuerName,
				"restricted": *a.serviceConfig.RestrictIssuer,
				"domains":    cluster.Shoot.Spec.DNS.Domain,
			},
			"issuers": issuers,
			"configuration": map[string]interface{}{
				"propagationTimeout": propagationTimeout,
			},
			"dnsChallengeOnShoot":              dnsChallengeOnShoot,
			"shootIssuers":                     shootIssuers,
			"genericTokenKubeconfigSecretName": extensions.GenericTokenKubeconfigSecretNameFromCluster(cluster),
		}
	)

	if err := gutil.NewShootAccessSecret(v1alpha1.ShootAccessSecretName, namespace).Reconcile(ctx, a.client); err != nil {
		return err
	}
	certManagementConfig["shootClusterSecret"] = gutil.SecretNamePrefixShootAccess + v1alpha1.ShootAccessSecretName

	cfg := certManagementConfig["configuration"].(map[string]interface{})
	if a.serviceConfig.DefaultRequestsPerDayQuota != nil {
		cfg["defaultRequestsPerDayQuota"] = *a.serviceConfig.DefaultRequestsPerDayQuota
	}

	if a.serviceConfig.ACME.PrecheckNameservers != nil {
		cfg["precheckNameservers"] = *a.serviceConfig.ACME.PrecheckNameservers
	}
	if certConfig.PrecheckNameservers != nil {
		servers := *certConfig.PrecheckNameservers
		if a.serviceConfig.ACME.PrecheckNameservers != nil {
			servers = mergeServers(servers, *a.serviceConfig.ACME.PrecheckNameservers)
		}
		cfg["precheckNameservers"] = servers
	}
	if a.serviceConfig.ACME.CACertificates != nil {
		cfg["caCertificates"] = *a.serviceConfig.ACME.CACertificates
	}
	if a.serviceConfig.ACME.DeactivateAuthorizations != nil {
		cfg["deactivateAuthorizations"] = *a.serviceConfig.ACME.DeactivateAuthorizations
	}

	if certConfig.Alerting != nil && certConfig.Alerting.CertExpirationAlertDays != nil {
		cfg["certExpirationAlertDays"] = *certConfig.Alerting.CertExpirationAlertDays
	}

	certManagementConfig, err = chart.InjectImages(certManagementConfig, imagevector.ImageVector(), []string{v1alpha1.CertManagementImageName})
	if err != nil {
		return fmt.Errorf("failed to find image version for %s: %v", v1alpha1.CertManagementImageName, err)
	}

	renderer, err := chartrenderer.NewForConfig(a.config)
	if err != nil {
		return fmt.Errorf("could not create chart renderer: %w", err)
	}

	a.logger.Info("Component is being applied", "component", "cert-management", "namespace", namespace)

	return a.createManagedResource(ctx, namespace, v1alpha1.CertManagementResourceNameSeed, "seed", renderer, v1alpha1.CertManagementChartNameSeed, namespace, certManagementConfig, nil)
}

func (a *actuator) createShootResources(ctx context.Context, certConfig *service.CertConfig, cluster *controller.Cluster, namespace string) error {
	dnsChallengeOnShoot, err := createDNSChallengeOnShootValues(certConfig.DNSChallengeOnShoot)
	if err != nil {
		return err
	}

	shootIssuers := a.createShootIssuersValues(certConfig)

	values := map[string]interface{}{
		"dnsChallengeOnShoot":           dnsChallengeOnShoot,
		"shootIssuers":                  shootIssuers,
		"shootAccessServiceAccountName": v1alpha1.ShootAccessServiceAccountName,
	}

	renderer, err := util.NewChartRendererForShoot(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("could not create chart renderer: %w", err)
	}

	return a.createManagedResource(ctx, namespace, v1alpha1.CertManagementResourceNameShoot, "", renderer, v1alpha1.CertManagementChartNameShoot, metav1.NamespaceSystem, values, nil)
}

func (a *actuator) deleteSeedResources(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for seed", "namespace", namespace)

	if err := kutil.DeleteObject(ctx, a.client, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: gutil.SecretNamePrefixShootAccess + v1alpha1.ShootAccessSecretName, Namespace: namespace}}); err != nil {
		return err
	}

	if err := managedresources.Delete(ctx, a.client, namespace, v1alpha1.CertManagementResourceNameSeed, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, v1alpha1.CertManagementResourceNameSeed)
}

func (a *actuator) deleteShootResources(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for shoot", "namespace", namespace)
	if err := managedresources.Delete(ctx, a.client, namespace, v1alpha1.CertManagementResourceNameShoot, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, v1alpha1.CertManagementResourceNameShoot)
}

func (a *actuator) createManagedResource(ctx context.Context, namespace, name, class string, renderer chartrenderer.Interface, chartName, chartNamespace string, chartValues map[string]interface{}, injectedLabels map[string]string) error {
	chartPath := filepath.Join(charts.ChartsPath, chartName)
	chart, err := renderer.RenderEmbeddedFS(charts.Internal, chartPath, chartName, chartNamespace, chartValues)
	if err != nil {
		return err
	}

	data := map[string][]byte{chartName: chart.Manifest()}
	keepObjects := false
	forceOverwriteAnnotations := false
	return managedresources.Create(ctx, a.client, namespace, name, nil, false, class, data, &keepObjects, injectedLabels, &forceOverwriteAnnotations)
}

func (a *actuator) updateStatus(ctx context.Context, ex *extensionsv1alpha1.Extension, certConfig *service.CertConfig) error {
	var resources []gardencorev1beta1.NamedResourceReference
	for _, issuerConfig := range certConfig.Issuers {
		name := "extension-shoot-cert-service-issuer-" + issuerConfig.Name
		resources = append(resources, gardencorev1beta1.NamedResourceReference{
			Name: name,
			ResourceRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Secret",
				Name:       name,
				APIVersion: "v1",
			},
		})
	}

	patch := client.MergeFrom(ex.DeepCopy())
	ex.Status.Resources = resources
	return a.client.Status().Patch(ctx, ex, patch)
}

func (a *actuator) createShootIssuersValues(certConfig *service.CertConfig) map[string]interface{} {
	shootIssuersEnabled := false
	if certConfig.ShootIssuers != nil {
		shootIssuersEnabled = certConfig.ShootIssuers.Enabled
	} else if a.serviceConfig.ShootIssuers != nil {
		shootIssuersEnabled = a.serviceConfig.ShootIssuers.Enabled
	}
	return map[string]interface{}{
		"enabled": shootIssuersEnabled,
	}
}

func mergeServers(serversList ...string) string {
	existing := map[string]struct{}{}
	merged := []string{}
	for _, servers := range serversList {
		for _, item := range strings.Split(servers, ",") {
			if _, ok := existing[item]; !ok {
				existing[item] = struct{}{}
				merged = append(merged, item)
			}
		}
	}
	return strings.Join(merged, ",")
}
