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
}

// IssuerConfig contains information for certificate issuers.
type IssuerConfig struct {
	Name   string
	Server string
	Email  string
}

// DNSChallengeOnShoot is used to create DNS01 challenges on shoot and not on seed.
type DNSChallengeOnShoot struct {
	Enabled   bool
	Namespace string
	DNSClass  *string
}
