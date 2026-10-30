// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
)

var _ = Describe("createValues RestrictIssuer gate", func() {
	const namespace = "shoot--foo--bar"

	var (
		ctx = context.Background()

		scheme  *runtime.Scheme
		c       client.Client
		cluster *extensionscontroller.Cluster
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(extensionscontroller.AddToScheme(scheme)).To(Succeed())

		c = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		cluster = &extensionscontroller.Cluster{
			Shoot: &gardencorev1beta1.Shoot{
				Spec: gardencorev1beta1.ShootSpec{
					DNS: &gardencorev1beta1.DNS{Domain: ptr.To("foo.example.com")},
				},
			},
		}
	})

	// Regression guard: RestrictIssuer must gate on the config flag, not on RestrictedIssuer()
	// (which requires RestrictedDomains to already be set -> circular, dead control). See f001.
	It("populates RestrictedDomains from the shoot domain when RestrictIssuer is enabled", func() {
		a := &actuator{client: c, serviceConfig: config.Configuration{RestrictIssuer: ptr.To(true)}}

		values, err := a.createValues(ctx, logr.Discard(), &service.CertConfig{}, cluster, namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(values).NotTo(BeNil())
		Expect(values.RestrictedDomains).To(Equal("foo.example.com"))
		Expect(values.RestrictedIssuer()).To(BeTrue())
	})

	It("leaves RestrictedDomains empty when RestrictIssuer is disabled", func() {
		a := &actuator{client: c, serviceConfig: config.Configuration{RestrictIssuer: ptr.To(false)}}

		values, err := a.createValues(ctx, logr.Discard(), &service.CertConfig{}, cluster, namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(values).NotTo(BeNil())
		Expect(values.RestrictedDomains).To(BeEmpty())
	})
})
