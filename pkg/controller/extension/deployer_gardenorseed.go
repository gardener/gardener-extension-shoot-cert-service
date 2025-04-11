// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/cert/source"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
)

type getSecretFromVirtualGardenByRole func(ctx context.Context, gardenRole string) (*corev1.Secret, error)

func (d *deployer) DeployGardenOrSeedManagedResource(ctx context.Context, log logr.Logger, c client.Client, getter getSecretFromVirtualGardenByRole) error {
	if d.values.ShootDeployment {
		return fmt.Errorf("not supported for shoot deployment")
	}

	var objects []client.Object

	objects = append(objects, d.createServiceAccount())
	objects = append(objects, d.createCACertificatesConfigMap())
	issuerObjects, err := d.createIssuers()
	if err != nil {
		return err
	}
	objects = append(objects, issuerObjects...)
	objects = append(objects, d.createRole())
	objects = append(objects, d.createRoleBinding())
	objects = append(objects, d.createService())
	deployment, err := d.createDeployment()
	if err != nil {
		return err
	}
	objects = append(objects, deployment)
	objects = append(objects, d.createVPA())

	objects = append(objects, d.createShootRole())
	objects = append(objects, d.createShootRoleBinding())
	objects = append(objects, d.createShootClusterRole())
	objects = append(objects, d.createShootClusterRoleBinding())
	objects = append(objects, d.createNetworkPolicy())
	crds, err := d.getShootCRDs()
	if err != nil {
		return err
	}
	for _, crd := range crds {
		crd.GetAnnotations()[resourcesv1alpha1.KeepObject] = "true"
	}
	objects = append(objects, crds...)
	cert, secret, err := d.createIngressWildcardCertAndSecret(ctx, log, getter)
	if err != nil {
		return err
	}
	objects = append(objects, cert, secret)

	objects = removeNilObjects(objects)
	registry := newManagedResourceRegistry()
	data, err := registry.AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	keepObjects := false
	forceOverwriteAnnotations := false
	return managedresources.Create(ctx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed(), nil, false, v1beta1constants.SeedResourceManagerClass, data, &keepObjects, nil, &forceOverwriteAnnotations)
}

func (d *deployer) DeleteGardenOrSeedManagedResourceAndWait(ctx context.Context, c client.Client, timeout time.Duration) error {
	if err := managedresources.Delete(ctx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed(), false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed())
}

func (d *deployer) createIngressWildcardCertAndSecret(ctx context.Context, log logr.Logger, getter getSecretFromVirtualGardenByRole) (*certv1alpha1.Certificate, *corev1.Secret, error) {
	if d.values.CertClass != "seed" || d.values.SeedIngressDNSDomain == "" {
		return nil, nil, nil
	}

	labels := map[string]string{
		v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlaneWildcardCert,
		certificate.ManagedByLabel:  ControllerName + "-controller",
	}

	secret, err := d.createSecretForSeedIngressWildcardCertDNSChallenge(ctx, log, getter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup secret name for seed ingress wildcard cert DNS challenge: %w", err)
	}

	annotations := map[string]string{
		source.AnnotClass: "seed",
	}
	if secret != nil {
		annotations[source.AnnotDNSRecordProviderType] = secret.Annotations[gardenerutils.DNSProvider]
		annotations[source.AnnotDNSRecordSecretRef] = secret.Namespace + "/" + secret.Name
	}
	cert := &certv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ingress-wildcard-cert",
			Namespace:   v1beta1constants.GardenNamespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: certv1alpha1.CertificateSpec{
			CommonName:   ptr.To("*." + d.values.SeedIngressDNSDomain),
			SecretLabels: labels,
			SecretRef: &corev1.SecretReference{
				Name:      "ingress-wildcard-cert",
				Namespace: v1beta1constants.GardenNamespace,
			},
		},
	}
	return cert, secret, nil
}

func (d *deployer) createSecretForSeedIngressWildcardCertDNSChallenge(ctx context.Context, log logr.Logger, getter getSecretFromVirtualGardenByRole) (*corev1.Secret, error) {
	if d.values.DNSSecretRole == "" {
		// assuming not configured, as no DNSChallenges needed
		log.Info("Warning: No DNS challenge secret configured for seed ingress wildcard cert. This may be ok if a CA issuer is used.")
		return nil, nil
	}

	secret, err := getter(ctx, d.values.DNSSecretRole)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dns-challenge-secret",
			Namespace: d.values.Namespace,
			Annotations: map[string]string{
				gardenerutils.DNSProvider: secret.Annotations[gardenerutils.DNSProvider],
			},
		},
		Data: secret.Data,
	}, nil
}

func (d *deployer) createNetworkPolicy() *networkingv1.NetworkPolicy {
	if len(d.values.ExtensionConfig.InClusterACMEServerNamespaceMatchLabel) == 0 {
		return nil
	}
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "egress-from-cert-controller-manager-to-labelled-namespaces",
			Namespace: d.values.Namespace,
			Annotations: map[string]string{
				"configured-by": "certificateConfig.inClusterACMEServerNamespaceMatchLabel",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: d.values.ExtensionConfig.InClusterACMEServerNamespaceMatchLabel,
							},
						},
					},
				},
			},
		},
	}
}
