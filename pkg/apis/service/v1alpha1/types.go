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

package v1alpha1

import (
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CertManagementResourceNameSeed is the name for Cert-Management resources in the seed.
const CertManagementResourceNameSeed = "extension-shoot-cert-service-seed"

// CertManagementKubecfg is the name of the kubeconfig secret.
const CertManagementKubecfg = "extension-shoot-cert-service.kubecfg"

// CertManagementResourceNameShoot is the name for Cert-Management resources in the shoot.
const CertManagementResourceNameShoot = "extension-shoot-cert-service-shoot"

// CertManagementImageName is the name of the Cert-Management image in the image vector.
const CertManagementImageName = "cert-management"

// CertManagementUserName is the name of the user Cert-Broker uses to connect to the target cluster.
const CertManagementUserName = "gardener.cloud:system:cert-management"

// ChartsPath is the path to the charts
var ChartsPath = filepath.Join("charts", "internal")

// ShootAccessSecretName is the name of the shoot access secret in the seed.
const ShootAccessSecretName = "extension-shoot-cert-service"

// ShootAccessServiceAccountName is the name of the service account used for accessing the shoot.
const ShootAccessServiceAccountName = ShootAccessSecretName

// CertManagementChartNameSeed is the name of the chart for Cert-Management in the seed.
const CertManagementChartNameSeed = "shoot-cert-management-seed"

// CertManagementChartNameShoot is the name of the chart for Cert-Management in the shoot.
const CertManagementChartNameShoot = "shoot-cert-management-shoot"

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
