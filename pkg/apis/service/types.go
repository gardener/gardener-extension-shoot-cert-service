// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertConfig configuration resource
type CertConfig struct {
	metav1.TypeMeta

	// Issuers is the configuration for certificate issuers.
	Issuers []IssuerConfig

	// DNSChallengeOnShoot controls where the DNS entries for DNS01 challenges are created.
	// If not specified the DNS01 challenges are written to the control plane namespace on the seed.
	DNSChallengeOnShoot *DNSChallengeOnShoot

	// ShootIssuers contains enablement for issuers on shoot cluster
	// If specified, it overwrites the ShootIssuers settings of the service configuration.
	ShootIssuers *ShootIssuers

	// PrecheckNameservers is used to specify a comma-separated list of DNS servers for checking availability for DNS
	// challenge before calling ACME CA. Please consider to specify nameservers per issuer instead.
	PrecheckNameservers *string
}

// IssuerConfig contains information for certificate issuers.
type IssuerConfig struct {
	Name   string
	Server string
	Email  string
	// RequestsPerDayQuota sets quota for certificate requests per day
	RequestsPerDayQuota *int

	// PrivateKeySecretName is the secret name for the ACME private key.
	// If not provided, a new private key is generated.
	PrivateKeySecretName *string

	// ACMEExternalAccountBinding is a reference to a CA external account of the ACME server.
	ExternalAccountBinding *ACMEExternalAccountBinding

	// SkipDNSChallengeValidation marks that this issuer does not validate DNS challenges.
	// In this case no DNS entries/records are created for a DNS Challenge and DNS propagation
	// is not checked.
	SkipDNSChallengeValidation *bool

	// Domains optionally specifies domains allowed or forbidden for certificate requests
	Domains *DNSSelection

	// PrecheckNameservers overwrites the default precheck nameservers used for checking DNS propagation.
	// Format `host` or `host:port`, e.g. "8.8.8.8" same as "8.8.8.8:53" or "google-public-dns-a.google.com:53".
	PrecheckNameservers []string
}

// DNSChallengeOnShoot is used to create DNS01 challenges on shoot and not on seed.
type DNSChallengeOnShoot struct {
	Enabled   bool
	Namespace string
	DNSClass  *string
}

// DNSSelection is a restriction on the domains to be allowed or forbidden for certificate requests
type DNSSelection struct {
	// Include are domain names for which certificate requests are allowed (including any subdomains)
	Include []string
	// Exclude are domain names for which certificate requests are forbidden (including any subdomains)
	Exclude []string
}

// ACMEExternalAccountBinding is a reference to a CA external account of the ACME server.
type ACMEExternalAccountBinding struct {
	// keyID is the ID of the CA key that the External Account is bound to.
	KeyID string

	// KeySecretName is the secret name of the
	// Secret which holds the symmetric MAC key of the External Account Binding with data key 'hmacKey'.
	// The secret key stored in the Secret **must** be un-padded, base64 URL
	// encoded data.
	KeySecretName string
}

// ShootIssuers holds enablement for issuers on shoot cluster
// If specified, it overwrites the ShootIssuers settings of the service configuration.
type ShootIssuers struct {
	Enabled bool
}
