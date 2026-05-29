// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
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
