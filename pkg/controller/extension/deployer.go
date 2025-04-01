// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"strings"

	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	certv1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/client"
)

// Values holds the configuration and settings for the certificate service extension deployment.
type Values struct {
	ExtensionConfig                  config.Configuration
	CertConfig                       service.CertConfig
	Namespace                        string
	Image                            string
	GenericTokenKubeconfigSecretName string
	RestrictedDomains                string
	Resources                        []core.NamedResourceReference

	ShootDeployment bool
	CertClass       string
}

func (v Values) getLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     v.chartNameSeed(),
		"app.kubernetes.io/instance": v.chartNameSeed(),
	}
}

func (v Values) getSelectLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     v.chartNameSeed(),
		"app.kubernetes.io/instance": v.chartNameSeed(),
	}
}

func (v Values) shootClusterRoleName() string {
	if v.ShootDeployment {
		return "extensions.gardener.cloud:extension-shoot-cert-service:shoot"
	}

	return "extensions.gardener.cloud:extension-shoot-cert-service:" + v.CertClass
}

func (v Values) chartNameShoot() string {
	if v.ShootDeployment {
		return certv1alpha1.CertManagementChartNameShootShoot
	}

	if v.CertClass == "garden" {
		return certv1alpha1.CertManagementChartNameGarden
	}
	return certv1alpha1.CertManagementChartNameSeed
}

func (v Values) chartNameSeed() string {
	if v.ShootDeployment {
		return certv1alpha1.CertManagementChartNameShootSeed
	}

	if v.CertClass == "garden" {
		return certv1alpha1.CertManagementChartNameGarden
	}
	return certv1alpha1.CertManagementChartNameSeed
}

func (v Values) resourceNameGardenOrSeed() string {
	if v.CertClass == "garden" {
		return certv1alpha1.CertManagementResourceNameGarden
	}
	return certv1alpha1.CertManagementResourceNameSeed
}

func (v Values) instanceNameGardenOrSeed() string {
	return "cert-management-" + v.CertClass
}

func (v Values) shootNamespace() string {
	if v.ShootDeployment {
		return "kube-system"
	}

	return v.Namespace
}

func (v Values) fullName() string {
	return "cert-controller-manager"
}

func (v Values) restrictedIssuer() bool {
	return v.RestrictedDomains != "" && ptr.Deref(v.ExtensionConfig.RestrictIssuer, false)
}

func (v Values) precheckNameservers() string {
	precheckNameservers := ""
	if v.ExtensionConfig.ACME != nil {
		precheckNameservers = ptr.Deref(v.ExtensionConfig.ACME.PrecheckNameservers, "")
	}
	if v.CertConfig.PrecheckNameservers != nil {
		precheckNameservers = mergeServers(*v.CertConfig.PrecheckNameservers, precheckNameservers)
	}
	return precheckNameservers
}

func (v Values) caCertificates() string {
	if v.ExtensionConfig.ACME != nil {
		return ptr.Deref(v.ExtensionConfig.ACME.CACertificates, "")
	}
	if v.ExtensionConfig.CA.CACertificates != nil {
		return ptr.Deref(v.ExtensionConfig.CA.CACertificates, "")
	}
	return ""
}

func (v Values) certExpirationAlertDays() int {
	if v.CertConfig.Alerting != nil && v.CertConfig.Alerting.CertExpirationAlertDays != nil {
		return *v.CertConfig.Alerting.CertExpirationAlertDays
	}
	return defaultCertExpirationAlertDays
}

func (v Values) deactivateAuthorizations() bool {
	if v.ExtensionConfig.ACME == nil {
		return false
	}
	return ptr.Deref(v.ExtensionConfig.ACME.DeactivateAuthorizations, false)
}

func (v Values) propagationTimeout() string {
	if v.ExtensionConfig.ACME != nil && v.ExtensionConfig.ACME.PropagationTimeout != nil {
		return v.ExtensionConfig.ACME.PropagationTimeout.Duration.String()
	}
	return ""
}

func (v Values) priorityClassName() string {
	if v.ShootDeployment {
		return "gardener-system-200"
	}
	return "gardener-garden-system-100"
}

func (v Values) shootIssuersEnabled() bool {
	if !v.ShootDeployment {
		return false
	}
	if v.CertConfig.ShootIssuers != nil {
		return v.CertConfig.ShootIssuers.Enabled
	}
	return false
}

func (v Values) dnsChallengeOnShootEnabled() bool {
	return v.ShootDeployment && v.CertConfig.DNSChallengeOnShoot != nil && v.CertConfig.DNSChallengeOnShoot.Enabled
}

type deployer struct {
	values Values
}

func newDeployer(values Values) *deployer {
	return &deployer{
		values: values,
	}
}

func mergeServers(serversList ...string) string {
	existing := map[string]struct{}{}
	merged := []string{}
	for _, servers := range serversList {
		for _, item := range strings.Split(servers, ",") {
			if _, ok := existing[item]; !ok {
				existing[item] = struct{}{}
				merged = append(merged, item)
			}
		}
	}
	return strings.Join(merged, ",")
}

func newManagedResourceRegistry() *managedresources.Registry {
	return managedresources.NewRegistry(client.ClusterScheme, client.ClusterCodec, client.ClusterSerializer)
}
