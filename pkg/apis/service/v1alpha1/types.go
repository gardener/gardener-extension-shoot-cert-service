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
var ChartsPath = filepath.Join("controllers", "extension-shoot-cert-service", "charts", "internal")

// CertManagementChartNameSeed is the name of the chart for Cert-Management in the seed.
const CertManagementChartNameSeed = "shoot-cert-management-seed"

// CertManagementChartNameShoot is the name of the chart for Cert-Management in the shoot.
const CertManagementChartNameShoot = "shoot-cert-management-shoot"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertConfig infrastructure configuration resource
type CertConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Issuers is the configuration for certificate issuers.
	Issuers []IssuerConfig `json:"issuers,omitempty"`
}

// IssuerConfig contains information for certificate issuers.
type IssuerConfig struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Email  string `json:"email"`
}
