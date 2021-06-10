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
	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config/v1alpha1"

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
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`
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
}
