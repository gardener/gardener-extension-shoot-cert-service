// SPDX-FileCopyrightText: Contributors to the Gardener project
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
	securityv1alpha1 "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/seedmanagement/encoding"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	operatorclient "github.com/gardener/gardener/pkg/operator/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Shoot-Cert-Service Tests", func() {
	var (
		garden             = &operatorv1alpha1.Garden{ObjectMeta: metav1.ObjectMeta{Name: "local"}}
		seed               = &gardencorev1beta1.Seed{ObjectMeta: metav1.ObjectMeta{Name: "local"}}
		gardenlet          = &seedmanagementv1alpha1.Gardenlet{ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "garden"}}
		operatorExtension  = &operatorv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Name: "extension-shoot-cert-service"}}
		runtimeExtension   = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "garden-controlplane-cert-service"}}
		seedExtension      = &extensionsv1alpha1.Extension{ObjectMeta: metav1.ObjectMeta{Namespace: "garden", Name: "controlplane-cert-service"}}
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
		virtualScheme := runtime.NewScheme()
		Expect(operatorclient.AddVirtualSchemeToScheme(virtualScheme)).To(Succeed())
		Expect(securityv1alpha1.AddToScheme(virtualScheme)).To(Succeed())
		virtualClusterClient, err := kubernetes.NewClientFromSecret(ctx, runtimeClient, v1beta1constants.GardenNamespace, "gardener",
			kubernetes.WithDisabledCachedClient(),
			kubernetes.WithClientOptions(client.Options{Scheme: virtualScheme}),
		)
		Expect(err).NotTo(HaveOccurred())

		By("Patch Seed: Add extension")
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(seed), seed)).To(Succeed())
		seedPatch := client.MergeFrom(seed.DeepCopy())
		seed.Spec.Extensions = []gardencorev1beta1.Extension{
			{
				Type:           "controlplane-cert-service",
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
			Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		}

		By("Patch Garden: Add garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch := client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = []operatorv1alpha1.GardenExtension{
			{
				Type:           "controlplane-cert-service",
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
		waitForExtensionToBeReconciled(ctx, seedExtension)

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

		By("Check DNS Provider Secret for Seed Ingress TLS Certificate")
		Expect(seedCertificate.Annotations["cert.gardener.cloud/dnsrecord-secret-ref"]).To(Equal("garden/ingress-wildcard-cert-dns-provider"))
		checkDNSProviderSecretLabels(ctx, seedCertificate.Namespace, "ingress-wildcard-cert-dns-provider", BeEmpty())

		By("Patch Seed: use WorkloadIdentity for DNS provider")
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(seed), seed)).To(Succeed())
		originalSeedDNSProviderRef := seed.Spec.DNS.Provider.CredentialsRef.DeepCopy()

		wli := &securityv1alpha1.WorkloadIdentity{ObjectMeta: metav1.ObjectMeta{Namespace: "garden-local", Name: "local"}}
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(wli), wli)).To(Succeed())

		// Patch the Gardenlet config so the refallowlist controller sets the annotation on the WLI.
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(gardenlet), gardenlet)).To(Succeed())
		gardenletCfg, err := encoding.DecodeGardenletConfiguration(&gardenlet.Spec.Config, false)
		Expect(err).NotTo(HaveOccurred())
		originalCredentialsRef := gardenletCfg.SeedConfig.Spec.DNS.Provider.CredentialsRef.DeepCopy()
		gardenletCfg.SeedConfig.Spec.DNS.Provider.CredentialsRef = &corev1.ObjectReference{
			APIVersion: "security.gardener.cloud/v1alpha1",
			Kind:       "WorkloadIdentity",
			Name:       wli.Name,
			Namespace:  wli.Namespace,
		}
		encodedCfg, err := encoding.EncodeGardenletConfiguration(gardenletCfg)
		Expect(err).NotTo(HaveOccurred())
		gardenletPatch := client.MergeFrom(gardenlet.DeepCopy())
		gardenlet.Spec.Config = *encodedCfg
		Expect(virtualClusterClient.Client().Patch(ctx, gardenlet, gardenletPatch)).To(Succeed())
		Eventually(func(g Gomega) {
			g.Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(wli), wli)).To(Succeed())
			g.Expect(wli.Annotations[v1beta1constants.AnnotationSeedNames]).To(Equal(seed.Name))
		}).Should(Succeed())

		seedPatch = client.MergeFrom(seed.DeepCopy())
		seed.Spec.DNS.Provider.CredentialsRef = &corev1.ObjectReference{
			APIVersion: "security.gardener.cloud/v1alpha1",
			Kind:       "WorkloadIdentity",
			Name:       wli.Name,
			Namespace:  wli.Namespace,
		}
		Expect(virtualClusterClient.Client().Patch(ctx, seed, seedPatch)).To(Succeed())

		By("Trigger Seed Extension Reconcile")
		triggerExtensionReconcile(ctx, seedExtension)
		waitForExtensionToBeReconciled(ctx, seedExtension)

		By("Check DNS Provider WorkloadIdentity Secret for Seed Ingress TLS Certificate")
		checkDNSProviderSecretLabels(ctx, seedCertificate.Namespace, "ingress-wildcard-cert-dns-provider",
			HaveKeyWithValue("security.gardener.cloud/purpose", "workload-identity-token-requestor"))

		By("Patch Garden: Remove garden extension")
		Expect(runtimeClient.Get(ctx, client.ObjectKeyFromObject(garden), garden)).To(Succeed())
		patch = client.MergeFrom(garden.DeepCopy())
		garden.Spec.Extensions = nil
		Expect(runtimeClient.Patch(ctx, garden, patch)).To(Succeed())
		waitForGardenToBeReconciled(ctx, garden)

		By("Patch Seed: Remove extension")
		// Revert Gardenlet config so the refallowlist controller removes the WLI annotation.
		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(gardenlet), gardenlet)).To(Succeed())
		gardenletCfg, err = encoding.DecodeGardenletConfiguration(&gardenlet.Spec.Config, false)
		Expect(err).NotTo(HaveOccurred())
		gardenletCfg.SeedConfig.Spec.DNS.Provider.CredentialsRef = originalCredentialsRef
		encodedCfg, err = encoding.EncodeGardenletConfiguration(gardenletCfg)
		Expect(err).NotTo(HaveOccurred())
		gardenletPatch = client.MergeFrom(gardenlet.DeepCopy())
		gardenlet.Spec.Config = *encodedCfg
		Expect(virtualClusterClient.Client().Patch(ctx, gardenlet, gardenletPatch)).To(Succeed())

		Expect(virtualClusterClient.Client().Get(ctx, client.ObjectKeyFromObject(seed), seed)).To(Succeed())
		seedPatch = client.MergeFrom(seed.DeepCopy())
		seed.Spec.Extensions = nil
		seed.Spec.DNS.Provider.CredentialsRef = originalSeedDNSProviderRef
		Expect(virtualClusterClient.Client().Patch(ctx, seed, seedPatch)).To(Succeed())

		By("Check Operator Extension required status")
		waitForOperatorExtensionToBeReconciled(ctx, operatorExtension, gardencorev1beta1.ConditionFalse, gardencorev1beta1.ConditionFalse)

		By("Delete Extension")
		Expect(runtimeClient.Delete(ctx, operatorExtension)).To(Succeed())
		waitForOperatorExtensionToBeDeleted(ctx, operatorExtension)
	})
})
