// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("gardenCert#delete", func() {
	var (
		ctx    = context.Background()
		scheme = runtime.NewScheme()
		c      client.Client
		r      *gardenCert
		ex     *extensionsv1alpha1.Extension
	)

	BeforeEach(func() {
		Expect(certv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(extensionscontroller.AddToScheme(scheme)).To(Succeed())
		Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed())
		c = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		r = newGardenCert(c, logr.Discard())
		ex = &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "garden-shoot-cert-service",
				Namespace: "garden",
			},
		}
		Expect(c.Create(ctx, ex)).To(Succeed())
	})

	It("succeeds when the certificate does not exist", func() {
		Expect(r.delete(ctx, ex)).To(Succeed())
	})

	It("deletes certificate and clears hash annotation when extension is not being deleted", func() {
		ex.Annotations = map[string]string{GardenRelevantDataHashAnnotation: "somehash"}
		Expect(c.Update(ctx, ex)).To(Succeed())

		cert := &certv1alpha1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.newCertificate().Name,
				Namespace: r.newCertificate().Namespace,
				Labels:    map[string]string{ManagedByLabel: ManagedByValue},
			},
		}
		Expect(c.Create(ctx, cert)).To(Succeed())

		Expect(r.delete(ctx, ex)).To(Succeed())

		existing := &certv1alpha1.Certificate{}
		Expect(client.IgnoreNotFound(c.Get(ctx, client.ObjectKeyFromObject(cert), existing))).To(Succeed())
		Expect(existing.Name).To(BeEmpty())

		updated := &extensionsv1alpha1.Extension{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(ex), updated)).To(Succeed())
		Expect(updated.Annotations).NotTo(HaveKey(GardenRelevantDataHashAnnotation))
	})

	It("skips deletion when certificate is not managed by us", func() {
		cert := &certv1alpha1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.newCertificate().Name,
				Namespace: r.newCertificate().Namespace,
				Labels:    map[string]string{"some-other-label": "value"},
			},
		}
		Expect(c.Create(ctx, cert)).To(Succeed())

		Expect(r.delete(ctx, ex)).To(Succeed())

		existing := &certv1alpha1.Certificate{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(cert), existing)).To(Succeed())
	})

	It("deletes certificate without patching extension when extension has a deletion timestamp", func() {
		cert := &certv1alpha1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.newCertificate().Name,
				Namespace: r.newCertificate().Namespace,
				Labels:    map[string]string{ManagedByLabel: ManagedByValue},
			},
		}
		Expect(c.Create(ctx, cert)).To(Succeed())

		// Simulate deletion by setting DeletionTimestamp on the local object only;
		// fake client does not allow setting it directly via Update.
		exWithDeletion := ex.DeepCopy()
		now := metav1.Now()
		exWithDeletion.DeletionTimestamp = &now

		Expect(r.delete(ctx, exWithDeletion)).To(Succeed())

		existing := &certv1alpha1.Certificate{}
		Expect(client.IgnoreNotFound(c.Get(ctx, client.ObjectKeyFromObject(cert), existing))).To(Succeed())
		Expect(existing.Name).To(BeEmpty())
	})
})
