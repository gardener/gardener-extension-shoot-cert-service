// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/utils/managedresources"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
)

const (
	roleName = "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager"
)

var (
	//go:embed assets/cert.gardener.cloud_certificaterevocations.yaml
	crdRevocations string

	//go:embed assets/cert.gardener.cloud_certificates.yaml
	crdCertificates string

	//go:embed assets/cert.gardener.cloud_issuers.yaml
	crdIssuers string
)

func (d *deployer) DeployShootManagedResource(ctx context.Context, c client.Client) error {
	if d.values.InternalDeployment {
		return fmt.Errorf("not supported for internal deployment")
	}

	var objects []client.Object

	objects = append(objects, d.createShootRole())
	objects = append(objects, d.createShootRoleBinding())
	objects = append(objects, d.createShootClusterRole())
	objects = append(objects, d.createShootClusterRoleBinding())

	crds, err := d.getShootCRDs()
	if err != nil {
		return err
	}
	objects = append(objects, crds...)

	registry := newManagedResourceRegistry()
	data, err := registry.AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	keepObjects := false
	forceOverwriteAnnotations := false
	return managedresources.Create(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameShoot, nil, false, "", data, &keepObjects, nil, &forceOverwriteAnnotations)
}

func (d *deployer) DeleteShootManagedResourceAndWait(ctx context.Context, c client.Client, timeout time.Duration) error {
	if d.values.InternalDeployment {
		return fmt.Errorf("not supported for internal deployment")
	}

	if err := managedresources.Delete(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameShoot, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameShoot)
}

func (d *deployer) createShootRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: d.values.shootNamespace(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				Verbs:         []string{"get", "watch", "update"},
				ResourceNames: []string{"shoot-cert-service"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{"coordination.k8s.io"},
				Resources:     []string{"leases"},
				ResourceNames: []string{"shoot-cert-service"},
				Verbs:         []string{"get", "watch", "update"},
			},
		},
	}
}

func (d *deployer) createShootRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: d.values.shootNamespace(),
		},
		Subjects: []rbacv1.Subject{d.getServiceAccount()},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
}

func (d *deployer) createShootClusterRole() *rbacv1.ClusterRole {
	certResources := []string{"certificates", "certificates/status", "certificaterevocations", "certificaterevocations/status"}
	if d.values.shootIssuersEnabled() {
		certResources = append(certResources, "issuers", "issuers/status")
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.values.shootClusterRoleName(),
			Labels: d.getShootLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "update", "watch"},
			},
			{
				APIGroups: []string{"gateway.networking.k8s.io"},
				Resources: []string{"gateways", "httproutes"},
				Verbs:     []string{"get", "list", "update", "watch"},
			},
			{
				APIGroups: []string{"networking.istio.io"},
				Resources: []string{"gateways", "virtualservices"},
				Verbs:     []string{"get", "list", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get", "list", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"cert.gardener.cloud"},
				Resources: certResources,
				Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list", "update", "create", "watch"},
			},
		},
	}

	if d.values.dnsChallengeOnShootEnabled() {
		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			APIGroups: []string{"dns.gardener.cloud"},
			Resources: []string{"dnsentries"},
			Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
		})
	}
	return role
}

func (d *deployer) createShootClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.values.shootClusterRoleName(),
			Labels: d.getShootLabels(),
		},
		Subjects: []rbacv1.Subject{d.getServiceAccount()},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     d.values.shootClusterRoleName(),
		},
	}
}

func (d *deployer) getServiceAccount() rbacv1.Subject {
	subjectName := v1alpha1.ShootAccessServiceAccountName
	subjectNamespace := "kube-system"
	if d.values.InternalDeployment {
		subjectName = "internal-cert-management"
		subjectNamespace = d.values.Namespace
	}

	return rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      subjectName,
		Namespace: subjectNamespace,
	}
}

// getShootCRDs reads cert-mangement CRDs from embedded resources.
func (d *deployer) getShootCRDs() ([]client.Object, error) {
	var crds []client.Object

	items := []string{crdCertificates, crdRevocations}
	if d.values.shootIssuersEnabled() || d.values.InternalDeployment {
		items = append(items, crdIssuers)
	}
	for _, data := range items {
		crd, err := stringToCRD(data)
		if err != nil {
			return nil, err
		}
		crds = append(crds, crd)
	}
	return crds, nil
}

func (d *deployer) getShootLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": d.values.chartNameShoot(),
	}
}

func stringToCRD(data string) (*apiextensionsv1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal([]byte(data), crd); err != nil {
		return nil, err
	}
	return crd, nil
}
