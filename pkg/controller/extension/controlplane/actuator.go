// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component/extensions/dnsrecord"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/extension/shared"
)

const (
	// EnvSeedName is the environment variable for the seed name.
	EnvSeedName = "SEED_NAME"

	// ManagedByLabel is a label used to identify the owner of certificate and secret resources.
	ManagedByLabel = "service.cert.extensions.gardener.cloud/managed-by"
	// ManagedByValue is a value used to identify own managed certificate and secret resources.
	ManagedByValue = "gardener-extension-shoot-cert-service"

	// ExtensionClassLabel is a label used to identify the extension class of the certificate resources.
	ExtensionClassLabel = "service.cert.extensions.gardener.cloud/extension-class"

	// SecretNameGardenCert is the name of the secret used for storing the garden certificate.
	// This name is used for backwards compatibility.
	SecretNameGardenCert = "tls"
	// SecretNameControlPlaneCert is the name of the secret used for storing the control plane certificate.
	// This name is used for backwards compatibility.
	SecretNameControlPlaneCert = "ingress-wildcard-cert"
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

	gardenClient     client.Client
	gardenClientLock sync.Mutex
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	certConfig, err := a.certConfigDecoder.DecodeAndValidateProviderConfig(ex, nil)
	if err != nil {
		return err
	}

	values, err := a.createValues(certConfig, namespace, ex)
	if err != nil {
		return err
	}

	if err := a.createResourcesForGardenOrSeed(ctx, log, *values); err != nil {
		return err
	}

	generateControlPlaneCertificate := ptr.Deref(certConfig.GenerateControlPlaneCertificate, false)
	if isGardenDeployment(ex) {
		handler := newGardenCert(a.client, log)
		if generateControlPlaneCertificate {
			if err := handler.reconcile(ctx, ex); err != nil {
				return err
			}
		} else {
			if err := handler.delete(ctx, ex); err != nil {
				return err
			}
		}
	} else {
		handler := newControlPlaneCert(a.client, log)
		if generateControlPlaneCertificate {
			seedName := os.Getenv(EnvSeedName)
			gardenClient, err := a.getOrCreateGardenClient()
			if err != nil {
				return err
			}

			seed, err := a.fetchSeedFromVirtualGarden(ctx, gardenClient, seedName)
			if err != nil {
				return fmt.Errorf("failed to get seed %s: %w", seedName, err)
			}

			handler.domain = seed.Spec.Ingress.Domain
			handler.dnsProviderType = seed.Spec.DNS.Provider.Type
			handler.credentialsDeployFunc, err = getDNSProviderCredentialsDeployer(ctx, gardenClient, seed)
			if err != nil {
				return fmt.Errorf("failed to get DNS provider credentials deployer for seed %s: %w", seedName, err)
			}
			if err := handler.reconcile(ctx); err != nil {
				return err
			}
		} else {
			if err := handler.delete(ctx); err != nil {
				return err
			}
		}
	}

	return a.updateStatus(ctx, ex, certConfig)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	log.Info("Component is being deleted", "component", "cert-management", "namespace", ex.GetNamespace())
	if isGardenDeployment(ex) {
		if err := newGardenCert(a.client, log).delete(ctx, ex); err != nil {
			return err
		}
	}
	return a.deleteResourcesForGardenOrSeed(ctx, log, ex)
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
	certConfig *service.CertConfig,
	namespace string,
	ex *extensionsv1alpha1.Extension,
) (*shared.Values, error) {
	values := shared.Values{
		ExtensionConfig:  a.serviceConfig,
		CertConfig:       *certConfig,
		Namespace:        namespace,
		Resources:        nil,
		ShootDeployment:  false,
		GardenDeployment: isGardenDeployment(ex),
		Replicas:         1,
	}

	if err := setValuesForGardenOrSeed(ex, &values); err != nil {
		return nil, err
	}

	var err error
	values.Image, err = shared.PrepareCertManagementImage()
	if err != nil {
		return nil, err
	}

	return &values, nil
}

func (a *actuator) createResourcesForGardenOrSeed(ctx context.Context, log logr.Logger, values shared.Values) error {
	log.Info("Component is being applied", "component", "cert-management", "namespace", values.Namespace, "certclass", values.CertClass)
	return shared.NewDeployer(values).DeployGardenOrSeedManagedResource(ctx, a.client)
}

func (a *actuator) deleteResourcesForGardenOrSeed(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	values := shared.Values{Namespace: ex.GetNamespace(), ShootDeployment: false}
	if err := setValuesForGardenOrSeed(ex, &values); err != nil {
		return err
	}
	log.Info("Deleting managed resource for garden or seed", "namespace", values.Namespace, "certclass", values.CertClass)
	return shared.NewDeployer(values).DeleteGardenOrSeedManagedResourceAndWait(ctx, a.client, 2*time.Minute)
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

func (a *actuator) getOrCreateGardenClient() (client.Client, error) {
	a.gardenClientLock.Lock()
	defer a.gardenClientLock.Unlock()

	if a.gardenClient != nil {
		return a.gardenClient, nil
	}

	gardenClient, err := a.createGardenClient()
	if err != nil {
		return nil, err
	}
	a.gardenClient = gardenClient
	return a.gardenClient, nil
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
	if err := securityv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add gardencorev1beta1 scheme: %w", err)
	}
	return client.New(restConfig, client.Options{
		Scheme: scheme,
	})
}

func isGardenDeployment(ex *extensionsv1alpha1.Extension) bool {
	return extensionsv1alpha1helper.GetExtensionClassOrDefault(ex.Spec.Class) == extensionsv1alpha1.ExtensionClassGarden
}

func setValuesForGardenOrSeed(ex *extensionsv1alpha1.Extension, values *shared.Values) error {
	if isGardenDeployment(ex) {
		values.CertClass = "garden"
	} else {
		values.CertClass = "seed"
		// use the extension namespace for deployment of cert-manager-controller
		values.Namespace = os.Getenv(shared.EnvLeaderElectionNamespace)
	}
	return nil
}

func getDNSProviderCredentialsDeployer(ctx context.Context, gardenClient client.Reader, seed *gardencorev1beta1.Seed) (dnsrecord.CredentialsDeployFunc, error) {
	if dnsConfig := seed.Spec.DNS; dnsConfig.Provider != nil {
		credentials, err := kubernetesutils.GetCredentialsByObjectReference(ctx, gardenClient, *dnsConfig.Provider.CredentialsRef)
		if err != nil {
			return nil, err
		}
		return dnsrecord.CredentialsDeployerFromCredentials(credentials, seed), nil
	}
	return nil, nil
}
