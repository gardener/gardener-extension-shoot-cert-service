// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	configv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains information about the certificate service configuration.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// IssuerName is the name of the issuer.
	IssuerName string `json:"issuerName"`
	// RestrictIssuer restricts the ACME issuer to shoot related domains.
	// +optional
	RestrictIssuer *bool `json:"restrictIssuer,omitempty"`
	// DefaultRequestsPerDayQuota restricts the certificate requests per issuer (can be overriden in issuer spec)
	// +optional
	DefaultRequestsPerDayQuota *int32 `json:"defaultRequestsPerDayQuota,omitempty"`
	// ShootIssuers contains enablement for issuers on shoot cluster
	// +optional
	ShootIssuers *ShootIssuers `json:"shootIssuers,omitempty"`
	// ACME contains ACME related configuration.
	ACME ACME `json:"acme"`
	// HealthCheckConfig is the config for the health check controller.
	// +optional
	HealthCheckConfig *configv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`
}

// ShootIssuers holds enablement for issuers on shoot cluster
type ShootIssuers struct {
	Enabled bool `json:"enabled"`
}

// ACME holds information about the ACME issuer used for the certificate service.
type ACME struct {
	// Email is the e-mail address used for the ACME issuer.
	Email string `json:"email"`
	// Server is the server address used for the ACME issuer.
	Server string `json:"server"`
	// PrivateKey is the key used for the ACME issuer.
	// +optional
	PrivateKey *string `json:"privateKey,omitempty"`
	// PropagationTimeout is the timeout for DNS01 challenges.
	// +optional
	PropagationTimeout *metav1.Duration `json:"propagationTimeout,omitempty"`
	// PrecheckNameservers is used to specify a comma-separated list of DNS servers for checking availability for DNS
	// challenge before calling ACME CA
	// +optional
	PrecheckNameservers *string `json:"precheckNameservers,omitempty"`
	// CACertificates are custom root certificates to be made available for the cert-controller-manager
	// +optional
	CACertificates *string `json:"caCertificates,omitempty"`
	// DeactivateAuthorizations enables deactivation of authorizations after successful certificate request
	// +optional
	DeactivateAuthorizations *bool `json:"deactivateAuthorizations,omitempty"`
}
