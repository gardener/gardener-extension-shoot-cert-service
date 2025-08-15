// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/certman2/core"
	"github.com/gardener/cert-management/pkg/shared/legobridge"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deployer) DeployGardenOrSeedManagedResource(ctx context.Context, c client.Client) error {
	if d.values.ShootDeployment {
		return fmt.Errorf("not supported for shoot deployment")
	}

	var objects []client.Object

	objects = append(objects, d.createServiceAccount())
	objects = append(objects, d.createCACertificatesConfigMap())
	issuerObjects, issuers, err := d.createIssuers()
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

	objects = removeNilObjects(objects)
	registry := newManagedResourceRegistry()
	data, err := registry.AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	if err := d.validateIssuerSecrets(ctx, c, issuers); err != nil {
		return fmt.Errorf("failed to validate issuer secrets: %w", err)
	}

	keepObjects := false
	forceOverwriteAnnotations := false
	return managedresources.Create(ctx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed(), nil, false, v1beta1constants.SeedResourceManagerClass, data, &keepObjects, nil, &forceOverwriteAnnotations)
}

func (d *Deployer) DeleteGardenOrSeedManagedResourceAndWait(ctx context.Context, c client.Client, timeout time.Duration) error {
	if err := managedresources.Delete(ctx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed(), false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, c, d.values.Namespace, d.values.resourceNameGardenOrSeed())
}

func (d *Deployer) createNetworkPolicy() *networkingv1.NetworkPolicy {
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

func (d *Deployer) validateIssuerSecrets(ctx context.Context, c client.Client, issuers []Issuer) error {
	var errs []error
	for _, issuer := range issuers {
		if issuer.ACME != nil {
			if issuer.ACME.PrivateKeySecretName != "" {
				secret := &corev1.Secret{}
				if err := c.Get(ctx, client.ObjectKey{Namespace: d.values.Namespace, Name: issuer.ACME.PrivateKeySecretName}, secret); err != nil {
					errs = append(errs, fmt.Errorf("failed to read secret for issuer %s: %w", issuer.Name, err))
				}
				if err := legobridge.ValidatePrivateKeySecretDataKeys(secret.Data); err != nil {
					errs = append(errs, fmt.Errorf("failed to validate ACME private key secret for issuer %s: %w", issuer.Name, err))
				}
			}
			if issuer.ACME.ExternalAccountBinding != nil && issuer.ACME.ExternalAccountBinding.KeySecretName != "" {
				support, err := core.NewHandlerSupport("dummy", d.values.Namespace, 100)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to create support for EAB key secret validation: %w", err))
				}
				_, _, err = support.LoadEABHmacKey(ctx,
					c,
					core.NewIssuerKey(client.ObjectKey{Namespace: d.values.Namespace, Name: issuer.Name}, false),
					&v1alpha1.ACMESpec{
						ExternalAccountBinding: &v1alpha1.ACMEExternalAccountBinding{
							KeyID:        issuer.ACME.ExternalAccountBinding.KeyID,
							KeySecretRef: &corev1.SecretReference{Namespace: d.values.Namespace, Name: issuer.ACME.ExternalAccountBinding.KeySecretName},
						},
					})
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to validate EAB key secret for issuer %s: %w", issuer.Name, err))
				}
			}
		}
	}
	return errors.Join(errs...)
}
