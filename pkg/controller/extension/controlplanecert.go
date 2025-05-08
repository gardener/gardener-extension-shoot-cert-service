// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/cert/source"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconciler reconciles Gardens.
type controlPlaneCert struct {
	client client.Client
	log    logr.Logger

	domain               string
	dnsProviderType      string
	dnsProviderSecretRef *corev1.SecretReference
}

func newControlPlaneCert(client client.Client, log logr.Logger) *controlPlaneCert {
	return &controlPlaneCert{
		client: client,
		log:    log.WithName("controlplane-cert"),
	}
}

func (r *controlPlaneCert) reconcile(ctx context.Context) error {
	labels := map[string]string{
		v1beta1constants.GardenRole: v1beta1constants.GardenRoleControlPlaneWildcardCert,
		ManagedByLabel:              ManagedByValue,
	}

	annotations := map[string]string{
		source.AnnotClass: "seed",
	}
	if ref := r.dnsProviderSecretRef; ref != nil {
		annotations[source.AnnotDNSRecordProviderType] = r.dnsProviderType
		annotations[source.AnnotDNSRecordSecretRef] = ref.Namespace + "/" + ref.Name
	}
	cert := r.newCertificate()
	result, err := controllerutils.CreateOrGetAndMergePatch(ctx, r.client, cert, func() error {
		cert.Annotations = annotations
		cert.Labels = labels
		cert.Spec.CommonName = ptr.To("*." + r.domain)
		cert.Spec.SecretLabels = labels
		cert.Spec.SecretRef = &corev1.SecretReference{
			Name:      SecretNameControlPlaneCert,
			Namespace: v1beta1constants.GardenNamespace,
		}
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

	return nil
}

func (r *controlPlaneCert) delete(ctx context.Context) error {
	cert := r.newCertificate()
	if err := r.client.Delete(ctx, cert); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	r.log.Info("Deleted certificate", "name", cert.Name)

	return nil
}

func (r *controlPlaneCert) newCertificate() *certv1alpha1.Certificate {
	return &certv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretNameControlPlaneCert,
			Namespace: v1beta1constants.GardenNamespace,
		},
	}
}
