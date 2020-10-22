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

package config

import (
	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config"

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
	// ACME contains ACME related configuration.
	ACME ACME
	// HealthCheckConfig is the config for the health check controller.
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig
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
}
