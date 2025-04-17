// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/extensions"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	utilsimagevector "github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/imagevector"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"
)

const (
	// EnvSeedName is the environment variable for the seed name.
	EnvSeedName = "SEED_NAME"
	// EnvSeedIngressDNSDomain is the environment variable for the seed ingress DNS domain.
	EnvSeedIngressDNSDomain = "SEED_INGRESS_DNS_DOMAIN"
	// EnvSeedDNSDomainSecretRole is the environment variable for the seed DNS domain secret role.
	// This role is used to look up the DNS secret in the seed namespace on the virtual garden used for DNS Challenges
	EnvSeedDNSDomainSecretRole = "SEED_DNS_SECRET_ROLE" // #nosec G101 -- false positive
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.Configuration, extensionClasses []extensionsv1alpha1.ExtensionClass) extension.Actuator {
	return &actuator{
		client:           mgr.GetClient(),
		config:           mgr.GetConfig(),
		scheme:           mgr.GetScheme(),
		decoder:          serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		serviceConfig:    config,
		extensionClasses: extensionClasses,
	}
}

type actuator struct {
	client           client.Client
	config           *rest.Config
	scheme           *runtime.Scheme
	decoder          runtime.Decoder
	extensionClasses []extensionsv1alpha1.ExtensionClass

	serviceConfig config.Configuration
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	var (
		namespace         = ex.GetNamespace()
		isShootDeployment = isShootDeployment(ex)
		cluster           *extensions.Cluster
		err               error
	)

	if isShootDeployment {
		cluster, err = controller.GetCluster(ctx, a.client, namespace)
		if err != nil {
			return err
		}
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

	values, err := a.createValues(ctx, log, certConfig, cluster, namespace, ex)
	if err != nil {
		return err
	}

	if isShootDeployment {
		if !controller.IsHibernated(cluster) {
			if err := a.createShootResourcesForShoot(ctx, log, *values); err != nil {
				return err
			}
		}
		if err := a.createSeedResourcesForShoot(ctx, log, *values); err != nil {
			return err
		}
	} else {
		if err := a.createResourcesForGardenOrSeed(ctx, log, *values); err != nil {
			return err
		}
	}

	return a.updateStatus(ctx, ex, certConfig)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	log.Info("Component is being deleted", "component", "cert-management", "namespace", namespace)
	if !isShootDeployment(ex) {
		return a.deleteResourcesForGardenOrSeed(ctx, log, ex)
	}

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
	ex *extensionsv1alpha1.Extension,
) (*Values, error) {
	values := Values{
		ExtensionConfig: a.serviceConfig,
		CertConfig:      *certConfig,
		Namespace:       namespace,
		Resources:       nil,
		ShootDeployment: isShootDeployment(ex),
		Replicas:        1,
	}

	if values.ShootDeployment {
		values.Replicas = int32(controller.GetReplicas(cluster, 1)) // #nosec G115 -- replicas are always small integers
		if values.restrictedIssuer() {
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
	} else {
		setValuesForGardenOrSeed(ex, &values)
	}

	images, err := utilsimagevector.FindImages(imagevector.ImageVector(), []string{v1alpha1.CertManagementImageName})
	if err != nil {
		return nil, fmt.Errorf("failed to find image version for %s: %w", v1alpha1.CertManagementImageName, err)
	}
	image, ok := images[v1alpha1.CertManagementImageName]
	if !ok {
		return nil, fmt.Errorf("failed to find image version for %s", v1alpha1.CertManagementImageName)
	}
	values.Image = image.String()

	return &values, nil
}

func (a *actuator) createSeedResourcesForShoot(ctx context.Context, log logr.Logger, values Values) error {
	log.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace)
	return newDeployer(values).DeploySeedManagedResource(ctx, a.client)
}

func (a *actuator) createResourcesForGardenOrSeed(ctx context.Context, log logr.Logger, values Values) error {
	log.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace, "certclass", values.CertClass)
	return newDeployer(values).DeployGardenOrSeedManagedResource(ctx, log, a.client, a.fetchSecretFromVirtualGardenByGardenRole)
}

func (a *actuator) createShootResourcesForShoot(ctx context.Context, log logr.Logger, values Values) error {
	if !values.ShootDeployment {
		return nil
	}
	log.Info("Creating managed resource for shoot", "namespace", values.Namespace)
	return newDeployer(values).DeployShootManagedResource(ctx, a.client)
}

func (a *actuator) deleteResourcesForGardenOrSeed(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	values := Values{Namespace: ex.GetNamespace(), ShootDeployment: false}
	setValuesForGardenOrSeed(ex, &values)
	log.Info("Deleting managed resource for garden or seed", "namespace", values.Namespace, "certclass", values.CertClass)
	return newDeployer(values).DeleteGardenOrSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
}

func (a *actuator) deleteSeedResourcesForShoot(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("Deleting managed resource for seed", "namespace", namespace)
	return newDeployer(Values{Namespace: namespace, ShootDeployment: true}).DeleteSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
}

func (a *actuator) deleteShootResourcesForShoot(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("Deleting managed resource for shoot", "namespace", namespace)
	return newDeployer(Values{Namespace: namespace, ShootDeployment: true}).DeleteShootManagedResourceAndWait(ctx, a.client, 2*time.Minute)
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

func (a *actuator) fetchSecretFromVirtualGardenByGardenRole(ctx context.Context, role string) (*corev1.Secret, error) {
	if role == "" {
		// assuming not configured, as no DNSChallenges needed
		return nil, nil
	}

	gardenClient, err := a.createGardenClient()
	if err != nil {
		return nil, err
	}
	seedNamespace := gardenerutils.ComputeGardenNamespace(os.Getenv(EnvSeedName))
	secretList := &corev1.SecretList{}
	if err := gardenClient.List(ctx, secretList, client.InNamespace(seedNamespace), client.MatchingLabels{v1beta1constants.GardenRole: role}); err != nil {
		return nil, fmt.Errorf("failed to list secrets in seed: %w", err)
	}
	if len(secretList.Items) == 0 {
		return nil, fmt.Errorf("no secret found in seed namespace %s with role %s", seedNamespace, role)
	}
	if len(secretList.Items) > 1 {
		return nil, fmt.Errorf("multiple secrets found in seed namespace %s with role %s", seedNamespace, role)
	}
	return &secretList.Items[0], nil
}

func (a *actuator) createGardenClient() (client.Client, error) {
	restConfig, err := kubernetes.RESTConfigFromKubeconfigFile(gardenerutils.PathGenericGardenKubeconfig, kubernetes.AuthTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read garden kubeconfig: %w", err)
	}
	return client.New(restConfig, client.Options{
		Scheme: a.scheme,
	})
}

func isShootDeployment(ex *extensionsv1alpha1.Extension) bool {
	return extensionsv1alpha1helper.GetExtensionClassOrDefault(ex.Spec.Class) == extensionsv1alpha1.ExtensionClassShoot
}

func isGardenDeployment(ex *extensionsv1alpha1.Extension) bool {
	return extensionsv1alpha1helper.GetExtensionClassOrDefault(ex.Spec.Class) == extensionsv1alpha1.ExtensionClassGarden
}

func setValuesForGardenOrSeed(ex *extensionsv1alpha1.Extension, values *Values) {
	if isGardenDeployment(ex) {
		values.CertClass = "garden"
	} else {
		values.CertClass = "seed"
		values.SeedIngressDNSDomain = os.Getenv(EnvSeedIngressDNSDomain)
		values.DNSSecretRole = os.Getenv(EnvSeedDNSDomainSecretRole)
		// use the extension namespace for deployment of cert-manager-controller
		values.Namespace = os.Getenv("LEADER_ELECTION_NAMESPACE")
	}
}
