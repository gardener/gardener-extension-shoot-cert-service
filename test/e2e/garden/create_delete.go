// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package garden

import (
	"context"
	"strings"
	"time"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Shoot-Cert-Service Tests", func() {
	var (
		garden             = &operatorv1alpha1.Garden{ObjectMeta: metav1.ObjectMeta{Name: "local"}}
		operatorExtension  = &operatorv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Name: "extension-shoot-cert-service"}}
		runtimeExtension   = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "shoot-cert-service"}}
		seedExtension      = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Name: "shoot-cert-service"}}
		runtimeCertificate = &certv1alpha1.Certificate{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "tls"}}
		seedCertificate    = &certv1alpha1.Certificate{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "ingress-wildcard-cert"}}
	)

	It("Create, Delete", Label("simple"), func() {
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()

		By("Deploy Extension")
		Expect(execMake(ctx, "extension-up")).To(Succeed())

		By("Patch Garden: Add new domain")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		newDomainName := strings.ReplaceAll(garden.Spec.VirtualCluster.DNS.Domains[0].Name, "virtual-garden", "vg")
		found := false
		for _, domain := range garden.Spec.VirtualCluster.DNS.Domains {
			if domain.Name == newDomainName {
				found = true
				break
			}
		}
		if !found {
			patch := client.MergeFrom(garden.DeepCopy())
			garden.Spec.VirtualCluster.DNS.Domains = append(garden.Spec.VirtualCluster.DNS.Domains, operatorv1alpha1.DNSDomain{
				Name:     newDomainName,
				Provider: garden.Spec.VirtualCluster.DNS.Domains[0].Provider,
			})
			Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		}

		By("Patch Garden: Add garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch := client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = []operatorv1alpha1.GardenExtension{
			{Type: "shoot-cert-service"},
		}
		Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())

		waitForGardenToBeReconciled(ctx, garden)

		By("Check Operator Extension required status")
		waitForOperatorExtensionToBeReconciled(ctx, operatorExtension, gardencorev1beta1.ConditionTrue, gardencorev1beta1.ConditionTrue)

		By("Check Garden Runtime Extension")
		waitForExtensionToBeReconciled(ctx, runtimeExtension)

		By("Check Seed Extension")
		seedExtension.Namespace = getExtensionNamespace(ctx, "extension-shoot-cert-service")
		waitForExtensionToBeReconciled(ctx, runtimeExtension)

		By("Check Virtual Garden/Ingress TLS Certificate")
		waitForCertificateToBeReconciled(ctx, runtimeCertificate, MatchFields(IgnoreExtras, Fields{
			"State":    Equal("Error"), // Error as DNS Challenge is not possible in the test environment
			"DNSNames": Equal([]string{"*.virtual-garden.local.gardener.cloud", "*.vg.local.gardener.cloud", "*.ingress.runtime-garden.local.gardener.cloud"}),
			"Message":  PointTo(ContainSubstring("Failed check: DNS record propagation")),
		}))

		By("Check Seed Ingress TLS Certificate")
		waitForCertificateToBeReconciled(ctx, seedCertificate, MatchFields(IgnoreExtras, Fields{
			"State":      Equal("Error"), // Error as DNS Challenge is not possible in the test environment
			"CommonName": PointTo(Equal("*.ingress.local.seed.local.gardener.cloud")),
			"Message":    PointTo(ContainSubstring("Failed check: DNS record propagation")),
		}))

		By("Simulate Virtual Garden TLS Certificate Ready")
		Expect(createDummyTLSSecret(ctx, runtimeCertificate)).To(Succeed())
		patch = client.MergeFrom(runtimeCertificate.DeepCopy())
		runtimeCertificate.Annotations["service.cert.extensions.gardener.cloud/test-simulate-ready"] = "true"
		Expect(runtimeClient.Patch(ctx, runtimeCertificate, patch)).To(Succeed())

		By("Wait for Virtual Garden Kube API Server")
		waitForVirtualGardenKubeAPIServerPatched(ctx)

		By("Patch Garden: Remove garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch = client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = nil
		Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		waitForGardenToBeReconciled(ctx, garden)

		By("Prepare extension down")
		Expect(execMake(ctx, "prepare-extension-down")).To(Succeed())

		By("Check Operator Extension required status")
		waitForOperatorExtensionToBeReconciled(ctx, operatorExtension, gardencorev1beta1.ConditionFalse, gardencorev1beta1.ConditionFalse)

		By("Delete Extension")
		Expect(runtimeClient.Delete(ctx, operatorExtension)).To(Succeed())
		waitForOperatorExtensionToBeDeleted(ctx, operatorExtension)
	})
})
