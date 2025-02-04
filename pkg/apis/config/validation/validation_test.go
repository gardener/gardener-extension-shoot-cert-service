// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config/validation"
)

var _ = Describe("Validation", func() {
	var (
		validACME = &config.ACME{
			Email:  "john.doe@example.com",
			Server: "https://acme-v02.api.letsencrypt.org/directory",
		}
		validCA = func() *config.CA {
			pemCert, pemKey, err := createCA()
			Expect(err).ToNot(HaveOccurred())
			return &config.CA{
				Certificate:    pemCert,
				CertificateKey: pemKey,
			}
		}
		validCACerts = func() string {
			pemCert, _, err := createCA()
			Expect(err).ToNot(HaveOccurred())
			return pemCert + "\n" + pemCert
		}
	)

	DescribeTable("#ValidateConfiguration",
		func(config config.Configuration, match gomegatypes.GomegaMatcher) {
			err := validation.ValidateConfiguration(&config)
			Expect(err).To(match)
		},
		Entry("Empty configuration", config.Configuration{
			IssuerName: "",
			ACME:       &config.ACME{},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("issuerName"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.server"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.email"),
			})),
		)),
		Entry("Invalid ACME configuration", config.Configuration{
			IssuerName: "gardener",
			ACME: &config.ACME{
				Email:  "john.doe.com",
				Server: "acme-v02.api.letsencrypt.org/directory",
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.server"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.email"),
			})),
		)),
		Entry("Invalid precheck nameservers and caCertificates", config.Configuration{
			IssuerName: "gardener",
			ACME: &config.ACME{
				Email:               validACME.Email,
				Server:              validACME.Server,
				PrecheckNameservers: ptr.To("8.8.8.8,foo.com"),
				CACertificates:      ptr.To("blabla"),
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.precheckNameservers"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("acme.caCertificates"),
			})),
		)),
		Entry("Valid precheck nameservers and caCertificates", config.Configuration{
			IssuerName: "gardener",
			ACME: &config.ACME{
				Email:               validACME.Email,
				Server:              validACME.Server,
				PrecheckNameservers: ptr.To("8.8.8.8,172.11.22.253"),
				CACertificates:      ptr.To(validCACerts()),
			},
		}, BeEmpty()),
		Entry("Invalid DefaultRequestsPerDayQuota", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(0)),
			ACME:                       validACME,
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("defaultRequestsPerDayQuota"),
			})),
		)),
		Entry("Valid configuration", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(50)),
			ACME:                       validACME,
		}, BeEmpty()),
		Entry("Valid PrivateKeyDefaults", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(50)),
			ACME:                       validACME,
			PrivateKeyDefaults: &config.PrivateKeyDefaults{
				Algorithm: ptr.To("ECDSA"),
				SizeRSA:   ptr.To(2048),
				SizeECDSA: ptr.To(256),
			},
		}, BeEmpty()),
		Entry("Invalid PrivateKeyDefaults", config.Configuration{
			IssuerName:                 "gardener",
			DefaultRequestsPerDayQuota: ptr.To(int32(50)),
			ACME:                       validACME,
			PrivateKeyDefaults: &config.PrivateKeyDefaults{
				Algorithm: ptr.To("X"),
				SizeRSA:   ptr.To(999),
				SizeECDSA: ptr.To(444),
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("privateKeyDefaults.algorithm"),
				"Detail": Equal("algorithm must either be 'RSA' or 'ECDSA'"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("privateKeyDefaults.sizeRSA"),
				"Detail": Equal("size for RSA algorithm must either be '2048' or '3072' or '4096'"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("privateKeyDefaults.sizeECDSA"),
				"Detail": Equal("size for ECDSA algorithm must either be '256' or '384'"),
			})),
		)),
		Entry("Valid specification of CA", config.Configuration{
			IssuerName: "gardener",
			CA:         validCA(),
		}, BeEmpty()),
		Entry("Invalid CA certificate and private key", config.Configuration{
			IssuerName: "gardener",
			CA: &config.CA{
				Certificate:    "blabla",
				CertificateKey: "blabla",
			},
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("ca.certificate"),
				"Detail": Equal("invalid certificate: expected PEM format"),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("ca.certificateKey"),
				"Detail": Equal("invalid certificate private key: expected PEM format"),
			})),
		)),
		Entry("Invalid specification of both ACME and CA", config.Configuration{
			IssuerName: "gardener",
			ACME:       validACME,
			CA:         validCA(),
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("acme"),
				"Detail": Equal("only one of ACME or CA can be specified"),
			})),
		)),
		Entry("Invalid specification of none of ACME and CA", config.Configuration{
			IssuerName: "gardener",
			ACME:       nil,
			CA:         nil,
		}, ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("acme"),
				"Detail": Equal("at least one of ACME or CA must be specified"),
			})),
		)),
	)
})

func createCA() (string, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}))
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1234),
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 365),
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}
	pemCert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
	return pemCert, pemKey, nil
}
