// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("controlPlaneCert#delete", func() {
	var (
		ctx    = context.Background()
		scheme = runtime.NewScheme()
		c      client.Client
		r      *controlPlaneCert
	)

	BeforeEach(func() {
		Expect(certv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(kubernetesscheme.AddToScheme(scheme)).To(Succeed())
		c = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		r = newControlPlaneCert(c, logr.Discard())
	})

	It("succeeds when the certificate does not exist", func() {
		Expect(r.delete(ctx)).To(Succeed())
	})

	It("skips deletion when certificate is not managed by us", func() {
		cert := &certv1alpha1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretNameControlPlaneCert,
				Namespace: v1beta1constants.GardenNamespace,
				Labels:    map[string]string{"some-other-label": "value"},
			},
		}
		Expect(c.Create(ctx, cert)).To(Succeed())
		secret := r.newDNSProviderSecret()
		Expect(c.Create(ctx, secret)).To(Succeed())

		Expect(r.delete(ctx)).To(Succeed())

		existing := &certv1alpha1.Certificate{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(cert), existing)).To(Succeed())
		existingSecret := r.newDNSProviderSecret()
		Expect(c.Get(ctx, client.ObjectKeyFromObject(existingSecret), existingSecret)).To(Succeed())
	})

	It("deletes certificate when it is managed by us", func() {
		cert := &certv1alpha1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretNameControlPlaneCert,
				Namespace: v1beta1constants.GardenNamespace,
				Labels:    map[string]string{ManagedByLabel: ManagedByValue},
			},
		}
		Expect(c.Create(ctx, cert)).To(Succeed())

		Expect(r.delete(ctx)).To(Succeed())

		existing := &certv1alpha1.Certificate{}
		Expect(client.IgnoreNotFound(c.Get(ctx, client.ObjectKeyFromObject(cert), existing))).To(Succeed())
		Expect(existing.Name).To(BeEmpty())
	})
})
