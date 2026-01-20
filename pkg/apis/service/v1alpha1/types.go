// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertManagementResourceNameSeed is the name for Cert-Management resources in the seed.
const CertManagementResourceNameSeed = "extension-shoot-cert-service-seed"

// CertManagementResourceNameShoot is the name for Cert-Management resources in the shoot.
const CertManagementResourceNameShoot = "extension-shoot-cert-service-shoot"

// CertManagementResourceNameGarden is the name for Cert-Management resources on the Garden runtime cluster.
const CertManagementResourceNameGarden = "extension-shoot-cert-service-garden"

// CertManagementImageName is the name of the Cert-Management image in the image vector.
const CertManagementImageName = "cert-management"

// ShootAccessSecretName is the name of the shoot access secret in the seed.
const ShootAccessSecretName = "extension-shoot-cert-service"

// ShootAccessServiceAccountName is the name of the service account used for accessing the shoot.
const ShootAccessServiceAccountName = ShootAccessSecretName

// CertManagementChartNameShootSeed is the name of the chart for Cert-Management in the seed.
const CertManagementChartNameShootSeed = "shoot-cert-management-seed"

// CertManagementChartNameShootShoot is the name of the chart for Cert-Management in the shoot.
const CertManagementChartNameShootShoot = "shoot-cert-management-shoot"

// CertManagementChartNameGarden is the name of the chart for Cert-Management deployment on the Garden runtime cluster.
const CertManagementChartNameGarden = "cert-management-garden"

// CertManagementChartNameSeed is the name of the chart for Cert-Management deployment on the seed cluster.
const CertManagementChartNameSeed = "cert-management-seed"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertConfig configuration resource
type CertConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Issuers is the configuration for certificate issuers.
	Issuers []IssuerConfig `json:"issuers,omitempty"`

	// DNSChallengeOnShoot controls where the DNS entries for DNS01 challenges are created.
	// If not specified the DNS01 challenges are written to the control plane namespace on the seed.
	// +optional
	DNSChallengeOnShoot *DNSChallengeOnShoot `json:"dnsChallengeOnShoot,omitempty"`

	// ShootIssuers contains enablement for issuers on shoot cluster
	// If specified, it overwrites the ShootIssuers settings of the service configuration.
	// +optional
	ShootIssuers *ShootIssuers `json:"shootIssuers,omitempty"`

	// PrecheckNameservers is used to specify a comma-separated list of DNS servers for checking availability for DNS
	// challenge before calling ACME CA. Please consider to specify nameservers per issuer instead.
	// +optional
	PrecheckNameservers *string `json:"precheckNameservers,omitempty"`

	// Alerting contains configuration for alerting of certificate expiration.
	// +optional
	Alerting *Alerting `json:"alerting,omitempty"`

	// GenerateControlPlaneCertificate is a boolean flag to indicate if the control plane certificate should be generated.
	// This is only relevant for the Garden runtime or seed cluster.
	// If not specified, the default value is false.
	// +optional
	GenerateControlPlaneCertificate *bool `json:"generateControlPlaneCertificate,omitempty"`

	// DNSClass is the DNS class used for DNS entries created for DNS01 challenges.
	// +optional
	DNSClass *string `json:"dnsClass,omitempty"`
}

// Alerting contains configuration for alerting of certificate expiration.
type Alerting struct {
	// CertExpirationAlertDays are the number of days before the certificate expiration date an alert is triggered.
	// +optional
	CertExpirationAlertDays *int `json:"certExpirationAlertDays,omitempty"`
}

// IssuerConfig contains information for certificate issuers.
type IssuerConfig struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Email  string `json:"email"`
	// RequestsPerDayQuota sets quota for certificate requests per day
	// +optional
	RequestsPerDayQuota *int `json:"requestsPerDayQuota,omitempty"`

	// PrivateKeySecretName is the secret name for the ACME private key.
	// If not provided, a new private key is generated.
	// +optional
	PrivateKeySecretName *string `json:"privateKeySecretName,omitempty"`

	// ACMEExternalAccountBinding is a reference to a CA external account of the ACME server.
	// +optional
	ExternalAccountBinding *ACMEExternalAccountBinding `json:"externalAccountBinding,omitempty"`

	// SkipDNSChallengeValidation marks that this issuer does not validate DNS challenges.
	// In this case no DNS entries/records are created for a DNS Challenge and DNS propagation
	// is not checked.
	// +optional
	SkipDNSChallengeValidation *bool `json:"skipDNSChallengeValidation,omitempty"`

	// Domains optionally specifies domains allowed or forbidden for certificate requests
	// +optional
	Domains *DNSSelection `json:"domains,omitempty"`

	// PrecheckNameservers overwrites the default precheck nameservers used for checking DNS propagation.
	// Format `host` or `host:port`, e.g. "8.8.8.8" same as "8.8.8.8:53" or "google-public-dns-a.google.com:53".
	// +optional
	PrecheckNameservers []string `json:"precheckNameservers,omitempty"`
}

// DNSChallengeOnShoot is used to create DNS01 challenges on shoot and not on seed.
type DNSChallengeOnShoot struct {
	Enabled   bool   `json:"enabled"`
	Namespace string `json:"namespace"`
	// +optional
	DNSClass *string `json:"dnsClass,omitempty"`
}

// DNSSelection is a restriction on the domains to be allowed or forbidden for certificate requests
type DNSSelection struct {
	// Include are domain names for which certificate requests are allowed (including any subdomains)
	//+ optional
	Include []string `json:"include,omitempty"`
	// Exclude are domain names for which certificate requests are forbidden (including any subdomains)
	// + optional
	Exclude []string `json:"exclude,omitempty"`
}

// ACMEExternalAccountBinding is a reference to a CA external account of the ACME server.
type ACMEExternalAccountBinding struct {
	// keyID is the ID of the CA key that the External Account is bound to.
	KeyID string `json:"keyID"`

	// KeySecretName is the secret name of the
	// Secret which holds the symmetric MAC key of the External Account Binding with data key 'hmacKey'.
	// The secret key stored in the Secret **must** be un-padded, base64 URL
	// encoded data.
	KeySecretName string `json:"keySecretName"`
}

// ShootIssuers holds enablement for issuers on shoot cluster
// If specified, it overwrites the ShootIssuers settings of the service configuration.
type ShootIssuers struct {
	Enabled bool `json:"enabled"`
}
