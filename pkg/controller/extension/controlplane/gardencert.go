// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"fmt"
	"maps"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/cert/source"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type gardenCert struct {
	client client.Client
	log    logr.Logger
}

func newGardenCert(client client.Client, log logr.Logger) *gardenCert {
	return &gardenCert{
		client: client,
		log:    log.WithName("garden-cert"),
	}
}

func (r *gardenCert) reconcile(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	var (
		dnsNames   []string
		dnsNameSet = sets.NewString()
	)

	gardenList := &operatorv1alpha1.GardenList{}
	if err := r.client.List(ctx, gardenList); err != nil {
		return fmt.Errorf("failed to list gardens: %w", err)
	}
	if len(gardenList.Items) == 0 {
		return nil
	}
	if len(gardenList.Items) > 1 {
		return fmt.Errorf("multiple gardens found, only one is supported")
	}
	garden := &gardenList.Items[0]

	for _, domain := range garden.Spec.VirtualCluster.DNS.Domains {
		if !dnsNameSet.Has(domain.Name) {
			dnsNameSet.Insert(domain.Name)
			dnsNames = append(dnsNames, fmt.Sprintf("*.%s", domain.Name))
		}
	}

	for _, domain := range garden.Spec.RuntimeCluster.Ingress.Domains {
		if !dnsNameSet.Has(domain.Name) {
			dnsNameSet.Insert(domain.Name)
			dnsNames = append(dnsNames, fmt.Sprintf("*.%s", domain.Name))
		}
	}

	if len(dnsNames) == 0 {
		return r.delete(ctx, ex)
	}

	cert := r.newCertificate()
	result, err := controllerutils.CreateOrGetAndMergePatch(ctx, r.client, cert, func() error {
		cert.Spec.DNSNames = dnsNames
		cert.Spec.SecretLabels = map[string]string{
			v1beta1constants.GardenRole: v1beta1constants.GardenRoleGardenWildcardCert,
			ManagedByLabel:              ManagedByValue,
			ExtensionClassLabel:         string(extensionsv1alpha1.ExtensionClassGarden),
		}
		cert.Spec.SecretRef = &corev1.SecretReference{
			Name:      SecretNameGardenCert,
			Namespace: v1beta1constants.GardenNamespace,
		}
		if cert.Annotations == nil {
			cert.Annotations = map[string]string{}
		}
		cert.Annotations[source.AnnotClass] = "garden"
		if garden.Spec.DNS != nil {
			cert.Annotations[source.AnnotDNSRecordProviderType] = garden.Spec.DNS.Providers[0].Type
			cert.Annotations[source.AnnotDNSRecordSecretRef] = garden.Spec.DNS.Providers[0].SecretRef.Name
			cert.Annotations[source.AnnotDNSRecordClass] = string(extensionsv1alpha1.ExtensionClassGarden)
		}
		if cert.Labels == nil {
			cert.Labels = map[string]string{}
		}
		maps.Copy(cert.Labels, cert.Spec.SecretLabels)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update certificate: %w", err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		r.log.Info("Created certificate", "name", cert.Name)
	case controllerutil.OperationResultUpdated:
		r.log.Info("Updated certificate", "name", cert.Name)
	case controllerutil.OperationResultNone:
		r.log.Info("Certificate unchanged", "name", cert.Name)
	}

	return r.patchHashAnnotation(ctx, ex, garden)
}

func (r *gardenCert) patchHashAnnotation(ctx context.Context, ex *extensionsv1alpha1.Extension, garden *operatorv1alpha1.Garden) error {
	patch := client.MergeFrom(ex.DeepCopy())
	if garden != nil {
		hash := calcGardenRelevantDataHash(garden)

		if ex.Annotations == nil {
			ex.Annotations = map[string]string{}
		}
		ex.Annotations[GardenRelevantDataHashAnnotation] = hash
	} else {
		delete(ex.Annotations, GardenRelevantDataHashAnnotation)
	}
	return r.client.Patch(ctx, ex, patch)
}

func (r *gardenCert) delete(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	cert := r.newCertificate()
	if err := r.client.Delete(ctx, cert); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	r.log.Info("Deleted certificate", "name", cert.Name)

	if ex.DeletionTimestamp != nil {
		return nil
	}
	return r.patchHashAnnotation(ctx, ex, nil)
}

func (r *gardenCert) newCertificate() *certv1alpha1.Certificate {
	return &certv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretNameGardenCert,
			Namespace: v1beta1constants.GardenNamespace,
		},
	}
}
