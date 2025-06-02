// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/managedresources"
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
