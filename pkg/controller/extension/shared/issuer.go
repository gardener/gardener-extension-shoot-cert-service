// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

// Issuer is the configuration model for an ACME or CA issuer.
type Issuer struct {
	Name                string
	ACME                *ACME
	CA                  *CA
	RequestsPerDayQuota int
	PrecheckNameservers []string
}

// ACME is the model for ACME configuration.
type ACME struct {
	Email                      string
	PrivateKey                 *string
	Server                     string
	PrivateKeySecretName       string
	ExternalAccountBinding     *ExternalAccountBinding
	SkipDNSChallengeValidation bool
	Domains                    *Domains
}

// ExternalAccountBinding is the configuration model for ExternalAccountBinding.
type ExternalAccountBinding struct {
	KeyID         string
	KeySecretName string
}

// Domains is the configuration model for Domains.
type Domains struct {
	Include []string
	Exclude []string
}

// CA is the model for CA configuration.
type CA struct {
	Certificate    string
	CertificateKey string
}
