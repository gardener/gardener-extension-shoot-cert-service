// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	utilsimagevector "github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/imagevector"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"
)

// ActuatorName is the name of the Certificate Service actuator.
const ActuatorName = "shoot-cert-service-actuator"

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(mgr manager.Manager, config config.Configuration, extensionClass extensionsv1alpha1.ExtensionClass) extension.Actuator {
	return &actuator{
		client:         mgr.GetClient(),
		config:         mgr.GetConfig(),
		decoder:        serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		logger:         log.Log.WithName(ActuatorName),
		serviceConfig:  config,
		extensionClass: extensionClass,
	}
}

type actuator struct {
	client         client.Client
	config         *rest.Config
	decoder        runtime.Decoder
	extensionClass extensionsv1alpha1.ExtensionClass

	serviceConfig config.Configuration

	logger logr.Logger
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	var (
		namespace = ex.GetNamespace()
		// TODO(martinweindel) Use `extensionsv1alpha1helper.GetExtensionClassOrDefault(ex.Spec.ExtensionClass)` once the method is available with gardener v1.116.0
		isShootDeployment = gutil.IsShootNamespace(namespace)
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

	values, err := a.createValues(ctx, certConfig, cluster, namespace, isShootDeployment)
	if err != nil {
		return err
	}

	if isShootDeployment {
		if !controller.IsHibernated(cluster) {
			if err := a.createShootResourcesForShoot(ctx, *values); err != nil {
				return err
			}
		}
		if err := a.createSeedResourcesForShoot(ctx, *values); err != nil {
			return err
		}
	} else {
		if err := a.createResourcesForGardenOrSeed(ctx, *values); err != nil {
			return err
		}
	}

	return a.updateStatus(ctx, ex, certConfig)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	a.logger.Info("Component is being deleted", "component", "cert-management", "namespace", namespace)
	if !gutil.IsShootNamespace(namespace) {
		return a.deleteResourcesForGardenOrSeed(ctx, namespace)
	}

	if err := a.deleteShootResourcesForShoot(ctx, namespace); err != nil {
		return err
	}
	return a.deleteSeedResourcesForShoot(ctx, namespace)
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

func (a *actuator) createValues(ctx context.Context, certConfig *service.CertConfig, cluster *controller.Cluster, namespace string, isShootDeployment bool) (*Values, error) {
	values := Values{
		ExtensionConfig: a.serviceConfig,
		CertConfig:      *certConfig,
		Namespace:       namespace,
		Resources:       nil,
		ShootDeployment: isShootDeployment,
		Replicas:        1,
	}

	if values.ShootDeployment {
		values.Replicas = int32(controller.GetReplicas(cluster, 1)) // #nosec G115 -- replicas are always small integers
		if values.restrictedIssuer() {
			if cluster.Shoot.Spec.DNS == nil || cluster.Shoot.Spec.DNS.Domain == nil {
				a.logger.Info("no domain given for shoot %s/%s - aborting", cluster.Shoot.Name, cluster.Shoot.Namespace)
				return nil, nil
			}
			values.RestrictedDomains = *cluster.Shoot.Spec.DNS.Domain
		}

		if err := gutil.NewShootAccessSecret(v1alpha1.ShootAccessSecretName, namespace).Reconcile(ctx, a.client); err != nil {
			return nil, err
		}
		values.GenericTokenKubeconfigSecretName = extensions.GenericTokenKubeconfigSecretNameFromCluster(cluster)
		values.Resources = cluster.Shoot.Spec.Resources
	} else {
		values.CertClass = "seed"
		if a.extensionClass == extensionsv1alpha1.ExtensionClassGarden {
			values.CertClass = "garden"
		}
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

func (a *actuator) createSeedResourcesForShoot(ctx context.Context, values Values) error {
	a.logger.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace)
	return newDeployer(values).DeploySeedManagedResource(ctx, a.client)
}

func (a *actuator) createResourcesForGardenOrSeed(ctx context.Context, values Values) error {
	a.logger.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace)
	return newDeployer(values).DeployGardenOrSeedManagedResource(ctx, a.client)
}

func (a *actuator) createShootResourcesForShoot(ctx context.Context, values Values) error {
	if !values.ShootDeployment {
		return nil
	}
	return newDeployer(values).DeployShootManagedResource(ctx, a.client)
}

func (a *actuator) deleteResourcesForGardenOrSeed(ctx context.Context, namespace string) error {
	return newDeployer(Values{Namespace: namespace, ShootDeployment: false}).DeleteGardenOrSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
}

func (a *actuator) deleteSeedResourcesForShoot(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for seed", "namespace", namespace)
	return newDeployer(Values{Namespace: namespace, ShootDeployment: true}).DeleteSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
}

func (a *actuator) deleteShootResourcesForShoot(ctx context.Context, namespace string) error {
	a.logger.Info("Deleting managed resource for shoot", "namespace", namespace)
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
