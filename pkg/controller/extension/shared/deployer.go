// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"fmt"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	utilsimagevector "github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-cert-service/imagevector"
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
	Resources                        []gardencorev1beta1.NamedResourceReference

	ShootDeployment  bool
	GardenDeployment bool
	CertClass        string
	Replicas         int32
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

func (v Values) RestrictedIssuer() bool {
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
	if v.GardenDeployment {
		return "gardener-garden-system-100"
	}
	return "gardener-system-200"
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

type Deployer struct {
	values Values
}

func NewDeployer(values Values) *Deployer {
	return &Deployer{
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

func PrepareCertManagementImage() (string, error) {
	images, err := utilsimagevector.FindImages(imagevector.ImageVector(), []string{certv1alpha1.CertManagementImageName})
	if err != nil {
		return "", fmt.Errorf("failed to find image version for %s: %w", certv1alpha1.CertManagementImageName, err)
	}
	image, ok := images[certv1alpha1.CertManagementImageName]
	if !ok {
		return "", fmt.Errorf("failed to find image version for %s", certv1alpha1.CertManagementImageName)
	}
	return image.String(), nil
}
