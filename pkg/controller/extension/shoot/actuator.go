// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/extensions"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.Configuration, extensionClasses []extensionsv1alpha1.ExtensionClass) extension.Actuator {
	return &actuator{
		client:            mgr.GetClient(),
		config:            mgr.GetConfig(),
		scheme:            mgr.GetScheme(),
		serviceConfig:     config,
		extensionClasses:  extensionClasses,
		certConfigDecoder: shared.NewCertConfigDecoder(mgr),
	}
}

type actuator struct {
	certConfigDecoder shared.CertConfigDecoder
	client            client.Client
	config            *rest.Config
	scheme            *runtime.Scheme
	decoder           runtime.Decoder
	extensionClasses  []extensionsv1alpha1.ExtensionClass

	serviceConfig config.Configuration
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	certConfig, err := a.certConfigDecoder.DecodeAndValidateProviderConfig(ex, cluster)
	if err != nil {
		return err
	}

	values, err := a.createValues(ctx, log, certConfig, cluster, namespace)
	if err != nil {
		return err
	}

	if !controller.IsHibernated(cluster) {
		if err := a.createShootResourcesForShoot(ctx, log, *values); err != nil {
			return err
		}
	}
	if err := a.createSeedResourcesForShoot(ctx, log, *values); err != nil {
		return err
	}

	return a.updateStatus(ctx, ex, certConfig)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	log.Info("Component is being deleted", "component", "cert-management", "namespace", namespace)

	if err := a.deleteShootResourcesForShoot(ctx, log, namespace); err != nil {
		return err
	}
	return a.deleteSeedResourcesForShoot(ctx, log, namespace)
}

// ForceDelete the Extension resource.
func (a *actuator) ForceDelete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Delete(ctx, log, ex)
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

func (a *actuator) createValues(
	ctx context.Context,
	log logr.Logger,
	certConfig *service.CertConfig,
	cluster *controller.Cluster,
	namespace string,
) (*shared.Values, error) {
	values := shared.Values{
		ExtensionConfig: a.serviceConfig,
		CertConfig:      *certConfig,
		Namespace:       namespace,
		Resources:       nil,
		ShootDeployment: true,
		Replicas:        1,
	}

	values.Replicas = int32(controller.GetReplicas(cluster, 1)) // #nosec G115 -- replicas are always small integers
	if values.RestrictedIssuer() {
		if cluster.Shoot.Spec.DNS == nil || cluster.Shoot.Spec.DNS.Domain == nil {
			log.Info("no domain given for shoot %s/%s - aborting", cluster.Shoot.Name, cluster.Shoot.Namespace)
			return nil, nil
		}
		values.RestrictedDomains = *cluster.Shoot.Spec.DNS.Domain
	}

	if err := gardenerutils.NewShootAccessSecret(v1alpha1.ShootAccessSecretName, namespace).Reconcile(ctx, a.client); err != nil {
		return nil, err
	}
	values.GenericTokenKubeconfigSecretName = extensions.GenericTokenKubeconfigSecretNameFromCluster(cluster)
	values.Resources = cluster.Shoot.Spec.Resources

	var err error
	values.Image, err = shared.PrepareCertManagementImage()
	if err != nil {
		return nil, err
	}

	return &values, nil
}

func (a *actuator) createSeedResourcesForShoot(ctx context.Context, log logr.Logger, values shared.Values) error {
	log.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace)
	return shared.NewDeployer(values).DeploySeedManagedResource(ctx, a.client)
}

func (a *actuator) createShootResourcesForShoot(ctx context.Context, log logr.Logger, values shared.Values) error {
	log.Info("Creating managed resource for shoot", "namespace", values.Namespace)
	return shared.NewDeployer(values).DeployShootManagedResource(ctx, a.client)
}

func (a *actuator) deleteSeedResourcesForShoot(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("Deleting managed resource for seed", "namespace", namespace)
	return shared.NewDeployer(shared.Values{Namespace: namespace, ShootDeployment: true}).DeleteSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
}

func (a *actuator) deleteShootResourcesForShoot(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("Deleting managed resource for shoot", "namespace", namespace)
	return shared.NewDeployer(shared.Values{Namespace: namespace, ShootDeployment: true}).DeleteShootManagedResourceAndWait(ctx, a.client, 2*time.Minute)
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

func (a *actuator) createShootIssuersValues(certConfig *service.CertConfig) map[string]any {
	shootIssuersEnabled := false
	if certConfig.ShootIssuers != nil {
		shootIssuersEnabled = certConfig.ShootIssuers.Enabled
	} else if a.serviceConfig.ShootIssuers != nil {
		shootIssuersEnabled = a.serviceConfig.ShootIssuers.Enabled
	}
	return map[string]any{
		"enabled": shootIssuersEnabled,
	}
}

func (a *actuator) fetchSeedFromVirtualGarden(ctx context.Context, gardenClient client.Client, seedName string) (*gardencorev1beta1.Seed, error) {
	seed := &gardencorev1beta1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: seedName,
		},
	}
	if err := gardenClient.Get(ctx, client.ObjectKeyFromObject(seed), seed); err != nil {
		return nil, fmt.Errorf("failed to get seed %s: %w", seedName, err)
	}
	return seed, nil
}

func (a *actuator) fetchSeedSecret(ctx context.Context, gardenClient client.Client, seedName string, ref corev1.SecretReference) (*corev1.Secret, error) {
	seedNamespace := gardenerutils.ComputeGardenNamespace(seedName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: seedNamespace,
		},
	}
	if err := gardenClient.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", ref.Namespace, ref.Name, err)
	}
	return secret, nil
}

func (a *actuator) createGardenClient() (client.Client, error) {
	restConfig, err := kubernetes.RESTConfigFromKubeconfigFile(gardenerutils.PathGenericGardenKubeconfig, kubernetes.AuthTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read garden kubeconfig: %w", err)
	}
	scheme := runtime.NewScheme()
	if err := kubernetesscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add kubernetes scheme: %w", err)
	}
	if err := gardencorev1beta1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add gardencorev1beta1 scheme: %w", err)
	}
	return client.New(restConfig, client.Options{
		Scheme: scheme,
	})
}
