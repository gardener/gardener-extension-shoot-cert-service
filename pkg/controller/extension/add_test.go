// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	_ "embed"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	serviceinstall "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/install"
)

var _ = Describe("#mapGardenToExtension", func() {
	var (
		scheme = runtime.NewScheme()
		mgr    manager.Manager
		ctx    = context.Background()
		c      client.Client
		ex     *extensionsv1alpha1.Extension
		garden = &operatorv1alpha1.Garden{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		mapper handler.TypedMapFunc[*operatorv1alpha1.Garden, reconcile.Request]

		expectRequests = func(expectedCount int) {
			Expect(c.Create(ctx, ex)).To(Succeed())
			requests := mapper(ctx, garden)
			Expect(requests).To(HaveLen(expectedCount))
		}
	)

	BeforeEach(func() {
		Expect(extensionscontroller.AddToScheme(scheme)).To(Succeed())
		Expect(serviceinstall.AddToScheme(scheme)).To(Succeed())

		c = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		mgr = &test.FakeManager{
			Scheme: c.Scheme(),
			Client: c,
		}
		mapper = mapGardenToExtension(mgr, logr.Discard())
		ex = &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "garden-shoot-cert-service",
				Namespace: "garden",
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:  "shoot-cert-service",
					Class: ptr.To(extensionsv1alpha1.ExtensionClassGarden),
					ProviderConfig: &runtime.RawExtension{
						Raw: []byte(`{
  "apiVersion": "service.cert.extensions.gardener.cloud/v1alpha1",
  "kind": "CertConfig",
  "generateControlPlaneCertificate": true
}`),
					},
				},
			},
		}
	})

	It("should do no request if GenerateControlPlaneCertificate is false", func() {
		ex.Spec.ProviderConfig = nil
		expectRequests(0)
	})

	It("should return a request if GenerateControlPlaneCertificate is true", func() {
		expectRequests(1)
	})

	It("should return a request if class is mismatching", func() {
		ex.Spec.Class = ptr.To(extensionsv1alpha1.ExtensionClassSeed)
		expectRequests(0)
	})

	It("should return a request if type is mismatching", func() {
		ex.Spec.Type = "foo"
		expectRequests(0)
	})

	It("should return no request if GenerateControlPlaneCertificate is true, but hash is matching", func() {
		ex.Annotations = map[string]string{
			GardenRelevantDataHashAnnotation: calcGardenRelevantDataHash(garden),
		}
		expectRequests(0)
	})
})
