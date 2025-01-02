// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/controller/issuer/certificate"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	TLSCertAPIServerNamesAnnotation = "service.cert.extensions.gardener.cloud/tls-cert-apiserver-names"
	TLSCertRequestedAtAnnotation    = "service.cert.extensions.gardener.cloud/tls-cert-requested-at"
	TLSCertHashAnnotation           = "service.cert.extensions.gardener.cloud/tls-cert-hash"

	ManagedByLabel      = "service.cert.extensions.gardener.cloud/managed-by"
	ExtensionClassLabel = "service.cert.extensions.gardener.cloud/extension-class"
)

// Reconciler reconciles Gardens.
type Reconciler struct {
	RuntimeClientSet  kubernetes.Interface
	ControllerOptions controller.Options
	GardenNamespace   string
}

// Reconcile performs the main reconciliation logic.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	cert := &certv1alpha1.Certificate{}
	if err := r.RuntimeClientSet.Client().Get(ctx, request.NamespacedName, cert); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if cert.DeletionTimestamp != nil {
		if result, err := r.delete(ctx, log, cert); err != nil {
			return result, err
		}
		return reconcile.Result{}, nil
	}

	if result, err := r.reconcile(ctx, log, cert); err != nil {
		return result, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	cert *certv1alpha1.Certificate,
) (
	reconcile.Result,
	error,
) {
	if cert.Status.State != "Ready" {
		log.Info("Certificate is not ready yet")
		return reconcile.Result{}, nil
	}

	apiDNSNames := cert.GetAnnotations()[TLSCertAPIServerNamesAnnotation]
	certHash := cert.GetLabels()[certificate.LabelCertificateNewHashKey]

	secret := &corev1.Secret{}
	ns := cert.Namespace
	var secretName string
	if cert.Spec.SecretRef != nil {
		secretName = cert.Spec.SecretRef.Name
		if cert.Spec.SecretRef.Namespace != "" {
			ns = cert.Spec.SecretRef.Namespace
		}
	} else if cert.Spec.SecretName != nil {
		secretName = *cert.Spec.SecretName
	}
	if err := r.RuntimeClientSet.Client().Get(ctx, client.ObjectKey{Namespace: ns, Name: secretName}, secret); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get certificate secret %s: %w", secretName, err)
	}
	requestedAt := secret.GetAnnotations()[certificate.AnnotationRequestedAt]

	if err := r.updateVirtualGardenDeploymentAnnotation(ctx, log, apiDNSNames, requestedAt, certHash); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update virtual garden deployment annotations: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) delete(
	ctx context.Context,
	log logr.Logger,
	_ *certv1alpha1.Certificate,
) (
	reconcile.Result,
	error,
) {
	if err := r.updateVirtualGardenDeploymentAnnotation(ctx, log, "", "", ""); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update virtual garden deployment annotations: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) newCertificate() *certv1alpha1.Certificate {
	return &certv1alpha1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls",
			Namespace: r.GardenNamespace,
		},
	}
}

func (r *Reconciler) updateVirtualGardenDeploymentAnnotation(
	ctx context.Context,
	log logr.Logger,
	apiserverNames string,
	requestedAt string,
	certHash string,
) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "virtual-garden-kube-apiserver",
			Namespace: r.GardenNamespace,
		},
	}
	if err := r.RuntimeClientSet.Client().Get(ctx, client.ObjectKeyFromObject(deploy), deploy); err != nil {
		return fmt.Errorf("failed to get deployment %s: %w", deploy.Name, err)
	}

	if deploy.Annotations[TLSCertAPIServerNamesAnnotation] == apiserverNames &&
		deploy.Annotations[TLSCertRequestedAtAnnotation] == requestedAt &&
		deploy.Annotations[TLSCertHashAnnotation] == certHash {
		return nil
	}

	patch := client.MergeFrom(deploy.DeepCopy())
	if apiserverNames != "" {
		if deploy.Annotations == nil {
			deploy.Annotations = map[string]string{}
		}
		deploy.Annotations[TLSCertAPIServerNamesAnnotation] = apiserverNames
		deploy.Annotations[TLSCertRequestedAtAnnotation] = requestedAt
		deploy.Annotations[TLSCertHashAnnotation] = certHash
	} else {
		delete(deploy.Annotations, TLSCertAPIServerNamesAnnotation)
		delete(deploy.Annotations, TLSCertRequestedAtAnnotation)
		delete(deploy.Annotations, TLSCertHashAnnotation)
	}
	if err := r.RuntimeClientSet.Client().Patch(ctx, deploy, patch); err != nil {
		return fmt.Errorf("failed to patch virtual garden kube-apisever deployment annotations: %w", err)
	}
	log.Info("Updated deployment annotations", "name", deploy.Name, "apiserverNames", apiserverNames, "certHash", certHash)

	return nil
}
