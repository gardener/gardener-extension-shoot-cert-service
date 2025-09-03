// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
)

const (
	serverPortHttp     = 10258
	ingressCertWorkers = 5
	certWorkers        = 5
	issuerWorkers      = 2
	vpaUpdateMode      = "Auto"

	defaultAlgorithm               = "RSA"
	defaultSizeRSA                 = 3072
	defaultSizeECDSA               = 384
	defaultCertExpirationAlertDays = 15

	shootAccessSecretName = gutil.SecretNamePrefixShootAccess + v1alpha1.ShootAccessSecretName
)

var (
	vpaMinAllowed = resource.MustParse("20Mi")
)

//go:embed assets/cert-dashboard.json
var certDashboardJSON string

func (d *Deployer) DeploySeedManagedResource(ctx context.Context, c client.Client) error {
	if !d.values.ShootDeployment {
		return fmt.Errorf("only supported for shoot deployment")
	}

	var objects []client.Object

	objects = append(objects, d.createPodDisruptionBudget())
	objects = append(objects, d.createServiceAccount())
	objects = append(objects, d.createCACertificatesConfigMap())
	issuerObjects, issuers, err := d.createIssuers()
	if err != nil {
		return err
	}
	objects = append(objects, issuerObjects...)
	objects = append(objects, d.createRole())
	objects = append(objects, d.createRoleBinding())
	objects = append(objects, d.createService())
	deployment, err := d.createDeployment()
	if err != nil {
		return err
	}
	objects = append(objects, deployment)
	objects = append(objects, d.createVPA())
	objects = append(objects, d.createDashboardsConfigMap())
	objects = append(objects, d.createPrometheusRule())
	objects = append(objects, d.createServiceMonitor())

	objects = removeNilObjects(objects)
	registry := newManagedResourceRegistry()
	data, err := registry.AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	if err := d.validateIssuerSecrets(ctx, c, issuers); err != nil {
		return fmt.Errorf("failed to validate issuer secrets: %w", err)
	}

	keepObjects := false
	forceOverwriteAnnotations := false
	return managedresources.Create(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameSeed, nil, false, v1beta1constants.SeedResourceManagerClass, data, &keepObjects, nil, &forceOverwriteAnnotations)
}

func (d *Deployer) DeleteSeedManagedResourceAndWait(ctx context.Context, c client.Client, timeout time.Duration) error {
	if err := kutil.DeleteObject(ctx, c, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: shootAccessSecretName, Namespace: d.values.Namespace}}); err != nil {
		return err
	}

	if err := managedresources.Delete(ctx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameSeed, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, c, d.values.Namespace, v1alpha1.CertManagementResourceNameSeed)
}

func (d *Deployer) createCACertificatesConfigMap() *corev1.ConfigMap {
	certs := d.values.caCertificates()
	if certs == "" {
		return nil
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-controller-manager-ca-certificates",
			Namespace: d.values.Namespace,
			Labels:    d.values.getLabels(),
		},
		Data: map[string]string{
			"certs.pem": certs,
		},
	}
}

func (d *Deployer) createDashboardsConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-controller-manager-dashboards",
			Namespace: d.values.Namespace,
			Labels: map[string]string{
				"dashboard.monitoring.gardener.cloud/shoot": "true",
			},
		},
		Data: map[string]string{
			"cert-controller-manager-dashboard.json": certDashboardJSON,
		},
	}
}

func (d *Deployer) createDeployment() (*appsv1.Deployment, error) {
	labels := d.values.getLabels()
	if d.values.ShootDeployment {
		labels["high-availability-config.resources.gardener.cloud/type"] = "controller"
	}
	podLabels := map[string]string{
		"gardener.cloud/role":                                           "controlplane",
		"networking.gardener.cloud/to-dns":                              "allowed",
		"networking.gardener.cloud/to-private-networks":                 "allowed",
		"networking.gardener.cloud/to-public-networks":                  "allowed",
		"networking.gardener.cloud/to-runtime-apiserver":                "allowed",
		"networking.resources.gardener.cloud/to-kube-apiserver-tcp-443": "allowed",
	}
	for k, v := range d.values.getLabels() {
		podLabels[k] = v
	}

	issuerChecksum, err := d.issuersChecksum()
	if err != nil {
		return nil, err
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.values.fullName(),
			Namespace: d.values.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			RevisionHistoryLimit: ptr.To[int32](2),
			Replicas:             ptr.To(d.values.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.values.getSelectLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"checksum/issuers": issuerChecksum,
					},
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					PriorityClassName: d.values.priorityClassName(),
					Containers: []corev1.Container{
						{
							Name:            d.values.chartNameSeed(),
							Image:           d.values.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
							},
							VolumeMounts: d.volumeMounts(),
							Args:         d.args(),
							Env:          d.env(),
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: serverPortHttp,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt32(serverPortHttp),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("5m"),
									corev1.ResourceMemory: resource.MustParse("30Mi"),
								},
							},
						},
					},
					Volumes:            d.volumes(),
					ServiceAccountName: d.values.chartNameSeed(),
				},
			},
		},
	}, nil
}

func (d *Deployer) createPodDisruptionBudget() *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-controller-manager",
			Namespace: d.values.Namespace,
			Labels:    d.values.getLabels(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: ptr.To(intstr.FromInt32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.values.getSelectLabels(),
			},
			UnhealthyPodEvictionPolicy: ptr.To(policyv1.AlwaysAllow),
		},
	}
}

func (d *Deployer) createPrometheusRule() *monitoringv1.PrometheusRule {
	alertDays := d.values.certExpirationAlertDays()
	if alertDays == 0 {
		return nil
	}

	labels := d.values.getLabels()
	labels["prometheus"] = "shoot"

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot-cert-controller-manager",
			Namespace: d.values.Namespace,
			Labels:    d.getPrometheusLabels(),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name: "cert-controller-manager.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "SslCertificateWillExpireSoon",
						Expr:  intstr.FromString(fmt.Sprintf("((cert_management_cert_object_expire > 0) - time()) / 86400 <= %d", alertDays)),
						For:   ptr.To(monitoringv1.Duration("30m")),
						Labels: map[string]string{
							"service":    "cert-controller-manager",
							"severity":   "critical",
							"type":       "seed",
							"visibility": "operator",
						},
						Annotations: map[string]string{
							"description": fmt.Sprintf("Certificate in namespace %s will expire in less than %d days.", d.values.Namespace, alertDays),
							"summary":     fmt.Sprintf("TLS certificate will expire in less than %d days", alertDays),
						},
					},
				},
			}},
		},
	}
}

func (d *Deployer) createRole() *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "extensions.gardener.cloud:extension-shoot-cert-service",
			Namespace: d.values.Namespace,
			Labels:    d.values.getLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"gardener-extension-shoot-cert-service"},
				Verbs:         []string{"get", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create"},
			},
		},
	}

	if d.values.ShootDeployment {
		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			APIGroups: []string{"dns.gardener.cloud"},
			Resources: []string{"dnsentries"},
			Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
		})
	} else {
		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			APIGroups: []string{"extensions.gardener.cloud"},
			Resources: []string{"dnsrecords"},
			Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
		})
	}

	role.Rules = append(role.Rules,
		rbacv1.PolicyRule{
			APIGroups: []string{"cert.gardener.cloud"},
			Resources: []string{"issuers", "issuers/status"},
			Verbs:     []string{"get", "update", "list", "patch", "watch"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{"", "events.k8s.io"},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch"},
		},
	)
	return role
}

func (d *Deployer) createRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.values.chartNameSeed(),
			Namespace: d.values.Namespace,
			Labels:    d.values.getLabels(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "extensions.gardener.cloud:extension-shoot-cert-service",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      d.values.chartNameSeed(),
				Namespace: d.values.Namespace,
			},
		},
	}
}

func (d *Deployer) createService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cert-controller-manager",
			Namespace: d.values.Namespace,
			Annotations: map[string]string{
				"networking.resources.gardener.cloud/from-all-scrape-targets-allowed-ports": fmt.Sprintf(`[{"port":%d,"protocol":"TCP"}]`, serverPortHttp),
			},
			Labels: d.values.getLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     serverPortHttp,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: d.values.getSelectLabels(),
		},
	}
}

func (d *Deployer) createServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.values.chartNameSeed(),
			Namespace: d.values.Namespace,
			Labels:    d.values.getLabels(),
		},
		AutomountServiceAccountToken: ptr.To(false),
	}
}

func (d *Deployer) createServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot-cert-controller-manager",
			Namespace: d.values.Namespace,
			Labels:    d.getPrometheusLabels(),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: d.values.getSelectLabels(),
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "metrics",
					RelabelConfigs: []monitoringv1.RelabelConfig{
						{
							Action: "labelmap",
							Regex:  "__meta_kubernetes_service_label_(.+)",
						},
					},
					MetricRelabelConfigs: []monitoringv1.RelabelConfig{
						{
							SourceLabels: []monitoringv1.LabelName{"__name__"},
							Action:       "keep",
							Regex:        "^(cert_management_.+)$",
						},
					},
				},
			},
		},
	}
}

func (d *Deployer) createVPA() *vpaautoscalingv1.VerticalPodAutoscaler {
	if !d.values.ShootDeployment {
		return nil
	}

	return &vpaautoscalingv1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vpa", d.values.fullName()),
			Namespace: d.values.Namespace,
		},
		Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       d.values.fullName(),
			},
			UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
				UpdateMode: ptr.To(vpaautoscalingv1.UpdateMode(vpaUpdateMode)),
			},
			ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
				ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
					{
						ContainerName:       "*",
						ControlledValues:    ptr.To(vpaautoscalingv1.ContainerControlledValues("RequestsOnly")),
						ControlledResources: ptr.To([]corev1.ResourceName{corev1.ResourceMemory}),
						MinAllowed: corev1.ResourceList{
							corev1.ResourceMemory: vpaMinAllowed,
						},
					},
				},
			},
		},
	}
}

func (d *Deployer) env() []corev1.EnvVar {
	if d.values.caCertificates() == "" {
		return nil
	}
	return []corev1.EnvVar{
		{
			Name:  "LEGO_CA_SYSTEM_CERT_POOL",
			Value: "true",
		},
		{
			Name:  "LEGO_CA_CERTIFICATES",
			Value: "/var/run/cert-manager/certs/certs.pem",
		},
	}
}

func (d *Deployer) args() []string {
	args := []string{fmt.Sprintf("--name=%s", d.values.fullName())}
	if d.values.ShootDeployment {
		args = append(args,
			"--namespace=kube-system",
			"--source=/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig")
	} else {
		args = append(args,
			fmt.Sprintf("--namespace=%s", d.values.Namespace),
			fmt.Sprintf("--cert-class=%s", d.values.CertClass),
			"--use-dnsrecords=true",
		)
	}
	args = append(args,
		fmt.Sprintf("--issuer.issuer-namespace=%s", d.values.Namespace),
		fmt.Sprintf("--issuer.default-issuer=%s", d.values.ExtensionConfig.IssuerName))
	if quota := ptr.Deref(d.values.ExtensionConfig.DefaultRequestsPerDayQuota, 0); quota > 0 {
		args = append(args, fmt.Sprintf("--issuer.default-requests-per-day-quota=%d", quota))
	}
	if d.values.RestrictedIssuer() {
		args = append(args, fmt.Sprintf("--issuer.default-issuer-domain-ranges=%s", d.values.RestrictedDomains))
	}
	if nameservers := d.values.precheckNameservers(); nameservers != "" {
		args = append(args, fmt.Sprintf("--issuer.precheck-nameservers=%s", nameservers))
	}
	if d.values.dnsChallengeOnShootEnabled() {
		args = append(args,
			"--dns=/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig",
			fmt.Sprintf("--issuer.dns-namespace=%s", d.values.CertConfig.DNSChallengeOnShoot.Namespace))
		if class := ptr.Deref(d.values.CertConfig.DNSChallengeOnShoot.DNSClass, ""); class != "" {
			args = append(args, fmt.Sprintf("--issuer.dns-class=%s", class))
		}
	} else {
		args = append(args, fmt.Sprintf("--issuer.dns-namespace=%s", d.values.Namespace))
	}
	args = append(args,
		fmt.Sprintf("--server-port-http=%d", serverPortHttp),
		fmt.Sprintf("--ingress-cert.targets.pool.size=%d", ingressCertWorkers),
		fmt.Sprintf("--service-cert.targets.pool.size=%d", certWorkers),
		fmt.Sprintf("--issuer.default.pool.size=%d", issuerWorkers))
	if timeout := d.values.propagationTimeout(); timeout != "" {
		args = append(args, fmt.Sprintf("--propagation-timeout=%s", timeout))
	}
	if d.values.shootIssuersEnabled() {
		args = append(args, "--allow-target-issuers")
	}
	if d.values.deactivateAuthorizations() {
		args = append(args, "--acme-deactivate-authorizations")
	}
	args = append(args,
		"--lease-name=shoot-cert-service",
		"--lease-resource-lock=leases",
		"--kubeconfig.disable-deploy-crds",
		"--source.disable-deploy-crds",
		"--target.disable-deploy-crds")

	algorithm := defaultAlgorithm
	sizeRSA := defaultSizeRSA
	sizeECDSA := defaultSizeECDSA
	if d.values.ExtensionConfig.PrivateKeyDefaults != nil {
		algorithm = ptr.Deref(d.values.ExtensionConfig.PrivateKeyDefaults.Algorithm, algorithm)
		sizeRSA = ptr.Deref(d.values.ExtensionConfig.PrivateKeyDefaults.SizeRSA, sizeRSA)
		sizeECDSA = ptr.Deref(d.values.ExtensionConfig.PrivateKeyDefaults.SizeECDSA, sizeECDSA)
	}
	args = append(args,
		fmt.Sprintf("--default-private-key-algorithm=%s", algorithm),
		fmt.Sprintf("--default-rsa-private-key-size=%d", sizeRSA),
		fmt.Sprintf("--default-ecdsa-private-key-size=%d", sizeECDSA),
	)

	return args
}

func (d *Deployer) volumeMounts() []corev1.VolumeMount {
	var mounts []corev1.VolumeMount
	if d.values.ShootDeployment {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "kubeconfig",
			MountPath: "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig",
			ReadOnly:  true,
		})
	}
	if d.values.caCertificates() != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "ca-certificates",
			MountPath: "/var/run/cert-manager/certs",
			ReadOnly:  true,
		})
	}
	return mounts
}

func (d *Deployer) volumes() []corev1.Volume {
	var volumes []corev1.Volume
	if d.values.ShootDeployment {
		volumes = append(volumes, corev1.Volume{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					DefaultMode: ptr.To[int32](420),
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: d.values.GenericTokenKubeconfigSecretName,
								},
								Optional: ptr.To(false),
								Items: []corev1.KeyToPath{
									{
										Key:  "kubeconfig",
										Path: "kubeconfig",
									},
								},
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: shootAccessSecretName,
								},
								Optional: ptr.To(false),
								Items: []corev1.KeyToPath{
									{
										Key:  "token",
										Path: "token",
									},
								},
							},
						},
					},
				},
			},
		})
	}
	if d.values.caCertificates() != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "ca-certificates",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-ca-certificates", d.values.fullName()),
					},
				},
			},
		})
	}
	return volumes
}

func (d *Deployer) getPrometheusLabels() map[string]string {
	labels := d.values.getLabels()
	labels["prometheus"] = "shoot"
	return labels
}

func removeNilObjects(objects []client.Object) []client.Object {
	var out []client.Object
	for _, obj := range objects {
		if obj != nil {
			out = append(out, obj)
		}
	}
	return out
}
