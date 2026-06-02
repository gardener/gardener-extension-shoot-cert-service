// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("dnsServiceExtensionPredicate", func() {
	var (
		predicate dnsServiceExtensionPredicate

		makeExtension = func(name string, annotations map[string]string) *extensionsv1alpha1.Extension {
			return &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   "shoot--foo--bar",
					Annotations: annotations,
				},
			}
		}
	)

	Describe("Create", func() {
		It("should accept the shoot-dns-service Extension", func() {
			Expect(predicate.Create(event.TypedCreateEvent[*extensionsv1alpha1.Extension]{
				Object: makeExtension(dnsServiceExtensionName, nil),
			})).To(BeTrue())
		})

		It("should reject Extensions with a different name", func() {
			Expect(predicate.Create(event.TypedCreateEvent[*extensionsv1alpha1.Extension]{
				Object: makeExtension("shoot-other-service", nil),
			})).To(BeFalse())
		})

		It("should reject a nil object", func() {
			Expect(predicate.Create(event.TypedCreateEvent[*extensionsv1alpha1.Extension]{
				Object: nil,
			})).To(BeFalse())
		})
	})

	Describe("Update", func() {
		It("should reject Extensions with a different name", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension("shoot-other-service", nil),
				ObjectNew: makeExtension("shoot-other-service", map[string]string{useNextGenerationControllerAnnotation: "true"}),
			})).To(BeFalse())
		})

		It("should reject when the annotation is unchanged", func() {
			annotations := map[string]string{useNextGenerationControllerAnnotation: "true"}
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension(dnsServiceExtensionName, annotations),
				ObjectNew: makeExtension(dnsServiceExtensionName, annotations),
			})).To(BeFalse())
		})

		It("should accept when the annotation is added", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension(dnsServiceExtensionName, nil),
				ObjectNew: makeExtension(dnsServiceExtensionName, map[string]string{useNextGenerationControllerAnnotation: "true"}),
			})).To(BeTrue())
		})

		It("should accept when the annotation value changes", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension(dnsServiceExtensionName, map[string]string{useNextGenerationControllerAnnotation: "false"}),
				ObjectNew: makeExtension(dnsServiceExtensionName, map[string]string{useNextGenerationControllerAnnotation: "true"}),
			})).To(BeTrue())
		})

		It("should accept when the annotation is removed", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension(dnsServiceExtensionName, map[string]string{useNextGenerationControllerAnnotation: "true"}),
				ObjectNew: makeExtension(dnsServiceExtensionName, nil),
			})).To(BeTrue())
		})

		It("should treat a nil ObjectOld as no prior annotation", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: nil,
				ObjectNew: makeExtension(dnsServiceExtensionName, map[string]string{useNextGenerationControllerAnnotation: "true"}),
			})).To(BeTrue())
		})

		It("should reject a nil ObjectNew", func() {
			Expect(predicate.Update(event.TypedUpdateEvent[*extensionsv1alpha1.Extension]{
				ObjectOld: makeExtension(dnsServiceExtensionName, nil),
				ObjectNew: nil,
			})).To(BeFalse())
		})
	})

	Describe("Delete", func() {
		It("should always reject", func() {
			Expect(predicate.Delete(event.TypedDeleteEvent[*extensionsv1alpha1.Extension]{
				Object: makeExtension(dnsServiceExtensionName, nil),
			})).To(BeFalse())
		})
	})

	Describe("Generic", func() {
		It("should always reject", func() {
			Expect(predicate.Generic(event.TypedGenericEvent[*extensionsv1alpha1.Extension]{
				Object: makeExtension(dnsServiceExtensionName, nil),
			})).To(BeFalse())
		})
	})
})

var _ = Describe("mapDNSServiceExtensionToCertServiceExtension", func() {
	var (
		ctx     = context.Background()
		mapFunc = mapDNSServiceExtensionToCertServiceExtension()

		makeExtension = func(name, namespace string) *extensionsv1alpha1.Extension {
			return &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
		}
	)

	It("should map a shoot-dns-service Extension to a request for the shoot-cert-service Extension in the same namespace", func() {
		Expect(mapFunc(ctx, makeExtension(dnsServiceExtensionName, "shoot--foo--bar"))).To(Equal([]reconcile.Request{{
			NamespacedName: client.ObjectKey{
				Name:      Type,
				Namespace: "shoot--foo--bar",
			},
		}}))
	})

	It("should return nil for an Extension with a different name", func() {
		Expect(mapFunc(ctx, makeExtension("shoot-other-service", "shoot--foo--bar"))).To(BeNil())
	})

	It("should return nil for a nil Extension", func() {
		Expect(mapFunc(ctx, nil)).To(BeNil())
	})

	It("should preserve the namespace from the source Extension", func() {
		Expect(mapFunc(ctx, makeExtension(dnsServiceExtensionName, "shoot--other--ns"))).To(Equal([]reconcile.Request{{
			NamespacedName: client.ObjectKey{
				Name:      Type,
				Namespace: "shoot--other--ns",
			},
		}}))
	})
})

var _ = Describe("isNextGenDNSShootServiceEnabled", func() {
	const namespace = "shoot--foo--bar"

	var (
		ctx = context.Background()

		scheme *runtime.Scheme
		c      client.Client
		a      *actuator
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(extensionscontroller.AddToScheme(scheme)).To(Succeed())

		c = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		a = &actuator{client: c}
	})

	DescribeTable("should return whether the next-gen DNS controller is enabled",
		func(existing *extensionsv1alpha1.Extension, expected bool) {
			if existing != nil {
				Expect(c.Create(ctx, existing)).To(Succeed())
			}

			enabled, err := a.isNextGenDNSShootServiceEnabled(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(Equal(expected))
		},
		Entry("Extension does not exist", nil, false),
		Entry("Extension has no annotations",
			&extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{Name: dnsServiceExtensionName, Namespace: namespace},
			}, false),
		Entry("annotation value is 'false'",
			&extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:        dnsServiceExtensionName,
					Namespace:   namespace,
					Annotations: map[string]string{useNextGenerationControllerAnnotation: "false"},
				},
			}, false),
		Entry("annotation value is some other string",
			&extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:        dnsServiceExtensionName,
					Namespace:   namespace,
					Annotations: map[string]string{useNextGenerationControllerAnnotation: "True"},
				},
			}, false),
		Entry("annotation value is 'true'",
			&extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:        dnsServiceExtensionName,
					Namespace:   namespace,
					Annotations: map[string]string{useNextGenerationControllerAnnotation: "true"},
				},
			}, true),
		Entry("Extension with the same name lives in another namespace",
			&extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:        dnsServiceExtensionName,
					Namespace:   "shoot--other--ns",
					Annotations: map[string]string{useNextGenerationControllerAnnotation: "true"},
				},
			}, false),
	)
})
