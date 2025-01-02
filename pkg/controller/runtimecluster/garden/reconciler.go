// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package garden

import (
	"context"
	"fmt"
	"strings"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
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

	garden := &operatorv1alpha1.Garden{}
	if err := r.RuntimeClientSet.Client().Get(ctx, request.NamespacedName, garden); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	if err := r.ensureAtMostOneGardenExists(ctx); err != nil {
		log.Error(err, "Reconciliation prevented without automatic requeue")
		return reconcile.Result{}, nil
	}

	if garden.DeletionTimestamp != nil {
		if result, err := r.delete(ctx, log, garden); err != nil {
			return result, err
		}
		return reconcile.Result{}, nil
	}

	if result, err := r.reconcile(ctx, log, garden); err != nil {
		return result, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) ensureAtMostOneGardenExists(ctx context.Context) error {
	gardenList := &metav1.PartialObjectMetadataList{}
	gardenList.SetGroupVersionKind(operatorv1alpha1.SchemeGroupVersion.WithKind("GardenList"))
	if err := r.RuntimeClientSet.Client().List(ctx, gardenList, client.Limit(2)); err != nil {
		return err
	}

	if len(gardenList.Items) <= 1 {
		return nil
	}

	return fmt.Errorf("there can be at most one operator.gardener.cloud/v1alpha1.Garden resource in the system at a time")
}

func (r *Reconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	garden *operatorv1alpha1.Garden,
) (
	reconcile.Result,
	error,
) {
	var (
		apiServerNames []string
		dnsNames       []string
	)

	for _, domain := range garden.Spec.VirtualCluster.DNS.Domains {
		apiServerNames = append(apiServerNames, fmt.Sprintf("api.%s", domain.Name))
		dnsNames = append(dnsNames, fmt.Sprintf("*.%s", domain.Name))
	}

	for _, domain := range garden.Spec.RuntimeCluster.Ingress.Domains {
		dnsNames = append(dnsNames, fmt.Sprintf("*.%s", domain.Name))
	}

	if len(dnsNames) == 0 {
		return r.delete(ctx, log, garden)
	}

	cert := r.newCertificate()
	result, err := controllerutils.CreateOrGetAndMergePatch(ctx, r.RuntimeClientSet.Client(), cert, func() error {
		cert.Spec.DNSNames = dnsNames
		cert.Spec.SecretLabels = map[string]string{
			v1beta1constants.GardenRole:     v1beta1constants.GardenRoleControlPlaneWildcardCert,
			certificate.ManagedByLabel:      ControllerName + "-controller",
			certificate.ExtensionClassLabel: string(extensionsv1alpha1.ExtensionClassGarden),
		}
		cert.Spec.SecretRef = &corev1.SecretReference{
			Name:      "tls",
			Namespace: r.GardenNamespace,
		}
		if cert.Annotations == nil {
			cert.Annotations = map[string]string{}
		}
		cert.Annotations["cert.gardener.cloud/class"] = "garden"
		cert.Annotations[certificate.TLSCertAPIServerNamesAnnotation] = strings.Join(apiServerNames, ",")
		if cert.Labels == nil {
			cert.Labels = map[string]string{}
		}
		for k, v := range cert.Spec.SecretLabels {
			cert.Labels[k] = v
		}
		return nil
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create or update certificate: %w", err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("Created certificate", "name", cert.Name)
	case controllerutil.OperationResultUpdated:
		log.Info("Updated certificate", "name", cert.Name)
	case controllerutil.OperationResultNone:
		log.Info("Certificate unchanged", "name", cert.Name)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) delete(
	ctx context.Context,
	log logr.Logger,
	_ *operatorv1alpha1.Garden,
) (
	reconcile.Result,
	error,
) {
	cert := r.newCertificate()
	if err := r.RuntimeClientSet.Client().Delete(ctx, cert); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to delete certificate: %w", err)
	}
	log.Info("Deleted certificate", "name", cert.Name)

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
