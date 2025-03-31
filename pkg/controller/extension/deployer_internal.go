// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
)

func (d *deployer) DeployInternalManagedResource(ctx context.Context, c client.Client) error {
	if !d.values.InternalDeployment {
		return fmt.Errorf("only supported for internal deployment")
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
	return managedresources.Create(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameInternal, nil, false, v1beta1constants.SeedResourceManagerClass, data, &keepObjects, nil, &forceOverwriteAnnotations)
}

func (d *deployer) DeleteInternalManagedResourceAndWait(ctx context.Context, c client.Client, timeout time.Duration) error {
	if err := managedresources.Delete(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameInternal, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameInternal)
}
