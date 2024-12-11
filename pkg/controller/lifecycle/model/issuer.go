// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

type Issuer struct {
	Name                string
	ACME                *ACME
	CA                  *CA
	RequestsPerDayQuota int
	PrecheckNameservers []string
}

type ACME struct {
	Email                      string
	PrivateKey                 *string
	Server                     string
	PrivateKeySecretName       string
	ExternalAccountBinding     *ExternalAccountBinding
	SkipDNSChallengeValidation bool
	Domains                    *Domains
}

type ExternalAccountBinding struct {
	KeyID         string
	KeySecretName string
}

type Domains struct {
	Include []string
	Exclude []string
}

type CA struct {
	Certificate    string
	CertificateKey string
}
