// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package install_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/install"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
)

var _ = Describe("CertConfig decoding and conversion", func() {
	var (
		codecs  serializer.CodecFactory
		decoder runtime.Decoder
		encoder runtime.Encoder

		decode = func(data string) *service.CertConfig {
			certConfig := &service.CertConfig{}
			_, _, err := decoder.Decode([]byte(data), nil, certConfig)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			return certConfig
		}

		encodeToV1alpha1 = func(certConfig *service.CertConfig) map[string]any {
			data, err := runtime.Encode(encoder, certConfig)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			result := map[string]any{}
			ExpectWithOffset(1, json.Unmarshal(data, &result)).To(Succeed())
			return result
		}
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		install.Install(scheme)
		codecs = serializer.NewCodecFactory(scheme, serializer.EnableStrict)
		// same decoder setup as used for the extension provider config
		decoder = codecs.UniversalDecoder()
		encoder = codecs.LegacyCodec(v1alpha1.SchemeGroupVersion)
	})

	Describe("precheckAdditionalWait", func() {
		DescribeTable("rejects invalid duration values", func(value string) {
			_, _, err := decoder.Decode([]byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","precheckAdditionalWait":`+value+`}`), nil, &service.CertConfig{})
			Expect(err).To(HaveOccurred())
		},
			Entry("numeric seconds", `300`),
			Entry("malformed duration", `"5 minutes"`),
			Entry("overflowing duration", `"999999999999999999h"`),
		)

		It("should decode YAML and round-trip a positive duration", func() {
			certConfig := decode(`apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
kind: CertConfig
precheckAdditionalWait: 5m
`)
			Expect(certConfig.PrecheckAdditionalWait).NotTo(BeNil())
			Expect(certConfig.PrecheckAdditionalWait.Duration).To(Equal(5 * time.Minute))

			external := encodeToV1alpha1(certConfig)
			Expect(external).To(HaveKeyWithValue("apiVersion", "service.cert.extensions.gardener.cloud/v1alpha1"))
			Expect(external).To(HaveKeyWithValue("precheckAdditionalWait", "5m0s"))

			data, err := json.Marshal(external)
			Expect(err).NotTo(HaveOccurred())
			Expect(decode(string(data))).To(Equal(certConfig))
		})

		It("should preserve sub-second duration precision", func() {
			certConfig := decode(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","precheckAdditionalWait":"1500ms"}`)
			Expect(certConfig.PrecheckAdditionalWait).NotTo(BeNil())
			Expect(certConfig.PrecheckAdditionalWait.Duration).To(Equal(1500 * time.Millisecond))
			Expect(encodeToV1alpha1(certConfig)).To(HaveKeyWithValue("precheckAdditionalWait", "1.5s"))
		})

		It("should decode JSON and round-trip an explicit zero duration", func() {
			certConfig := decode(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","precheckAdditionalWait":"0s"}`)
			Expect(certConfig.PrecheckAdditionalWait).NotTo(BeNil())
			Expect(certConfig.PrecheckAdditionalWait.Duration).To(BeZero())

			external := encodeToV1alpha1(certConfig)
			Expect(external).To(HaveKeyWithValue("precheckAdditionalWait", "0s"))

			data, err := json.Marshal(external)
			Expect(err).NotTo(HaveOccurred())
			roundTripped := decode(string(data))
			Expect(roundTripped.PrecheckAdditionalWait).NotTo(BeNil())
			Expect(roundTripped.PrecheckAdditionalWait.Duration).To(BeZero())
		})

		It("should keep the field unset if not specified", func() {
			certConfig := decode(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig"}`)
			Expect(certConfig.PrecheckAdditionalWait).To(BeNil())

			Expect(encodeToV1alpha1(certConfig)).NotTo(HaveKey("precheckAdditionalWait"))
		})
	})
})
