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
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	operatorv1alpha1 "github.com/gardener/gardener/pkg/apis/operator/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	operatorclient "github.com/gardener/gardener/pkg/operator/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Shoot-Cert-Service Tests", func() {
	var (
		garden             = &operatorv1alpha1.Garden{ObjectMeta: metav1.ObjectMeta{Name: "local"}}
		seed               = &gardencorev1beta1.Seed{ObjectMeta: metav1.ObjectMeta{Name: "local"}}
		operatorExtension  = &operatorv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Name: "extension-shoot-cert-service"}}
		runtimeExtension   = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "shoot-cert-service"}}
		seedExtension      = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Name: "shoot-cert-service"}}
		runtimeCertificate = &certv1alpha1.Certificate{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "tls"}}
		seedCertificate    = &certv1alpha1.Certificate{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "ingress-wildcard-cert"}}

		rawExtension = &runtime.RawExtension{
			Raw: []byte(`{
  "apiVersion": "service.cert.extensions.gardener.cloud/v1alpha1",
  "kind": "CertConfig",
  "generateControlPlaneCertificate": true
}`),
		}
	)

	It("Create, Delete", Label("simple"), func() {
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()

		By("Deploy Extension")
		Expect(execMake(ctx, "extension-up")).To(Succeed())

		By("Get Virtual Garden Client")
		virtualClusterClient, err := kubernetes.NewClientFromSecret(ctx, runtimeClient, v1beta1constants.GardenNamespace, "gardener",
			kubernetes.WithDisabledCachedClient(),
			kubernetes.WithClientOptions(client.Options{Scheme: operatorclient.VirtualScheme}),
		)
		Expect(err).NotTo(HaveOccurred())

		By("Patch Seed: Add extension")
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(seed), seed)).To(Succeed())
		seedPatch := client.MergeFrom(seed.DeepCopy())
		seed.Spec.Extensions = []gardencorev1beta1.Extension{
			{
				Type:           "shoot-cert-service",
				ProviderConfig: rawExtension,
			},
		}
		Expect(virtualClusterClient.Client().Patch(ctx, seed, seedPatch)).To(Succeed())

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
			// Set SNI section to verify that extension reconciliation is performed even garden reconciliation fails as TLS secret is not yet existing.
			garden.Spec.VirtualCluster.Kubernetes.KubeAPIServer.SNI = &operatorv1alpha1.SNI{
				DomainPatterns: []string{"api." + newDomainName},
				SecretName:     "tls",
			}
			Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		}

		By("Patch Garden: Add garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch := client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = []operatorv1alpha1.GardenExtension{
			{
				Type:           "shoot-cert-service",
				ProviderConfig: rawExtension,
			},
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
			"State":    Equal("Ready"),
			"DNSNames": Equal([]string{"*.virtual-garden.local.gardener.cloud", "*.vg.local.gardener.cloud", "*.ingress.runtime-garden.local.gardener.cloud"}),
		}))

		By("Check Seed Ingress TLS Certificate")
		waitForCertificateToBeReconciled(ctx, seedCertificate, MatchFields(IgnoreExtras, Fields{
			"State":      Equal("Ready"),
			"CommonName": PointTo(Equal("*.ingress.local.seed.local.gardener.cloud")),
		}))

		By("Patch Garden: Remove garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch = client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = nil
		Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		waitForGardenToBeReconciled(ctx, garden)

		By("Patch Seed: Remove extension")
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(seed), seed)).To(Succeed())
		seedPatch = client.MergeFrom(seed.DeepCopy())
		seed.Spec.Extensions = nil
		Expect(virtualClusterClient.Client().Patch(ctx, seed, seedPatch)).To(Succeed())

		By("Check Operator Extension required status")
		waitForOperatorExtensionToBeReconciled(ctx, operatorExtension, gardencorev1beta1.ConditionFalse, gardencorev1beta1.ConditionFalse)

		By("Delete Extension")
		Expect(runtimeClient.Delete(ctx, operatorExtension)).To(Succeed())
		waitForOperatorExtensionToBeDeleted(ctx, operatorExtension)
	})
})
