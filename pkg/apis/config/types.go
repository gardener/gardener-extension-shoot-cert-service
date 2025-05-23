// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	extensionsconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains information about the certificate service configuration.
type Configuration struct {
	metav1.TypeMeta

	// IssuerName is the name of the issuer.
	IssuerName string
	// RestrictIssuer restricts the ACME issuer to shoot related domains.
	RestrictIssuer *bool
	// DefaultRequestsPerDayQuota restricts the certificate requests per issuer (can be overriden in issuer spec)
	DefaultRequestsPerDayQuota *int32
	// ShootIssuers contains enablement for issuers on shoot cluster
	ShootIssuers *ShootIssuers
	// ACME contains ACME related configuration.
	ACME *ACME
	// CA contains CA related configuration.
	CA *CA
	// HealthCheckConfig is the config for the health check controller.
	HealthCheckConfig *extensionsconfigv1alpha1.HealthCheckConfig
	// PrivateKeyDefaults default algorithm and sizes for certificate private keys.
	PrivateKeyDefaults *PrivateKeyDefaults
	// InClusterACMEServerNamespaceMatchLabel is the match label used to create a network policy to allow egress from the "cert-controller-manager" to a namespace with these labels.
	// It can be set to allow access to an in-cluster ACME server from the cert-controller-manager.
	InClusterACMEServerNamespaceMatchLabel map[string]string
}

// PrivateKeyDefaults default algorithm and sizes for certificate private keys.
type PrivateKeyDefaults struct {
	// Algorithm is the default algorithm ('RSA' or 'ECDSA')
	Algorithm *string
	// SizeRSA is the default size for RSA algorithm.
	SizeRSA *int
	// SizeECDSA is the default size for ECDSA algorithm.
	SizeECDSA *int
}

// ShootIssuers holds enablement for issuers on shoot cluster
type ShootIssuers struct {
	Enabled bool
}

// ACME holds information about the ACME issuer used for the certificate service.
type ACME struct {
	// Email is the e-mail address used for the ACME issuer.
	Email string
	// Server is the server address used for the ACME issuer.
	Server string
	// PrivateKey is the key used for the ACME issuer.
	PrivateKey *string
	// PropagationTimeout is the timeout for DNS01 challenges.
	PropagationTimeout *metav1.Duration
	// PrecheckNameservers is used to specify a comma-separated list of DNS servers for checking availability for DNS
	// challenge before calling ACME CA
	PrecheckNameservers *string
	// CACertificates are custom root certificates to be made available for the cert-controller-manager
	CACertificates *string
	// DeactivateAuthorizations enables deactivation of authorizations after successful certificate request
	DeactivateAuthorizations *bool
	// SkipDNSChallengeValidation skips the DNS challenge validation
	SkipDNSChallengeValidation *bool
}

type CA struct {
	// Certificate is the public certificate of the CA in PEM format.
	Certificate string
	// CertificateKey is the private certificate key of the CA in PEM format.
	CertificateKey string
}
