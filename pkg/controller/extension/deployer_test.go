// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	_ "embed"
	"reflect"
	"strings"
	"time"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	servicev1alpha1 "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/v1alpha1"
	certserviceclient "github.com/gardener/gardener-extension-shoot-cert-service/pkg/client"
)

var _ = Describe("deployer", func() {
	var (
		ctx       = context.Background()
		c         client.Client
		consistOf func(...client.Object) types.GomegaMatcher
		values    Values

		standardShootResources = func() []client.Object {
			return []client.Object{
				&apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "certificates.cert.gardener.cloud",
					},
				},
				&apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "certificaterevocations.cert.gardener.cloud",
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "extensions.gardener.cloud:extension-shoot-cert-service:shoot",
						Labels: map[string]string{
							"app.kubernetes.io/instance": "shoot-cert-management-shoot",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"networking.k8s.io"},
							Resources: []string{"ingresses"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{"gateway.networking.k8s.io"},
							Resources: []string{"gateways", "httproutes"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{"networking.istio.io"},
							Resources: []string{"gateways", "virtualservices"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"services"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"cert.gardener.cloud"},
							Resources: []string{"certificates", "certificates/status", "certificaterevocations", "certificaterevocations/status"},
							Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"events"},
							Verbs:     []string{"create", "patch"},
						},
						{
							APIGroups: []string{"apiextensions.k8s.io"},
							Resources: []string{"customresourcedefinitions"},
							Verbs:     []string{"get", "list", "update", "create", "watch"},
						},
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "extensions.gardener.cloud:extension-shoot-cert-service:shoot",
						Labels: map[string]string{
							"app.kubernetes.io/instance": "shoot-cert-management-shoot",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service:shoot",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "extension-shoot-cert-service",
							Namespace: "kube-system",
						},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
						Namespace: "kube-system",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"configmaps"},
							Verbs:     []string{"create"},
						},
						{
							APIGroups:     []string{""},
							Resources:     []string{"configmaps"},
							Verbs:         []string{"get", "watch", "update"},
							ResourceNames: []string{"shoot-cert-service"},
						},
						{
							APIGroups: []string{"coordination.k8s.io"},
							Resources: []string{"leases"},
							Verbs:     []string{"create"},
						},
						{
							APIGroups:     []string{"coordination.k8s.io"},
							Resources:     []string{"leases"},
							ResourceNames: []string{"shoot-cert-service"},
							Verbs:         []string{"get", "watch", "update"},
						},
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
						Namespace: "kube-system",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "extension-shoot-cert-service",
							Namespace: "kube-system",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
					},
				},
			}
		}

		deployment = func(namespace, certClass string, internal bool) *appsv1.Deployment {
			name := "shoot-cert-management-seed"
			shootNamespace := "kube-system"
			priorityClassName := "gardener-system-200"

			if internal {
				name = "cert-management-" + certClass
				shootNamespace = namespace
				priorityClassName = "gardener-garden-system-100"
			}

			args := []string{
				"--name=cert-controller-manager",
				"--namespace=" + shootNamespace,
			}
			if !internal {
				args = append(args, "--source=/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig")
			} else {
				args = append(args,
					"--issuer.cert-class="+certClass,
					"--use-dnsrecords=true",
				)
			}
			args = append(args,
				"--issuer.issuer-namespace="+namespace,
				"--issuer.default-issuer=garden",
				"--issuer.default-requests-per-day-quota=100",
				"--issuer.dns-namespace="+namespace,
				"--server-port-http=10258",
				"--ingress-cert.targets.pool.size=5",
				"--service-cert.targets.pool.size=5",
				"--issuer.default.pool.size=2",
				"--acme-deactivate-authorizations",
				"--lease-name=shoot-cert-service",
				"--lease-resource-lock=leases",
				"--kubeconfig.disable-deploy-crds",
				"--source.disable-deploy-crds",
				"--target.disable-deploy-crds",
				"--default-private-key-algorithm=RSA",
				"--default-rsa-private-key-size=3072",
				"--default-ecdsa-private-key-size=384",
			)

			obj := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert-controller-manager",
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":     name,
						"app.kubernetes.io/instance": name,
					},
				},
				Spec: appsv1.DeploymentSpec{
					RevisionHistoryLimit: ptr.To[int32](2),
					Replicas:             ptr.To[int32](1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name":     name,
							"app.kubernetes.io/instance": name,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"checksum/issuers": "61a61fbafea713a7ce63eeb2e6335d4199b8fc075e91d91cdfa9a86cee6f1afc",
							},
							Labels: map[string]string{
								"app.kubernetes.io/name":                                        name,
								"app.kubernetes.io/instance":                                    name,
								"gardener.cloud/role":                                           "controlplane",
								"networking.gardener.cloud/to-dns":                              "allowed",
								"networking.gardener.cloud/to-private-networks":                 "allowed",
								"networking.gardener.cloud/to-public-networks":                  "allowed",
								"networking.gardener.cloud/to-runtime-apiserver":                "allowed",
								"networking.resources.gardener.cloud/to-kube-apiserver-tcp-443": "allowed",
							},
						},
						Spec: corev1.PodSpec{
							PriorityClassName: priorityClassName,
							Containers: []corev1.Container{
								{
									Name:            name,
									Image:           "example.com/gardener-project/releases/cert-controller-manager:v0.0.0",
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: &corev1.SecurityContext{
										AllowPrivilegeEscalation: ptr.To(false),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											MountPath: "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig",
											Name:      "kubeconfig",
											ReadOnly:  true,
										},
										{
											Name:      "ca-certificates",
											MountPath: "/var/run/cert-manager/certs",
											ReadOnly:  true,
										},
									},
									Args: args,
									Env: []corev1.EnvVar{
										{
											Name:  "LEGO_CA_SYSTEM_CERT_POOL",
											Value: "true",
										},
										{
											Name:  "LEGO_CA_CERTIFICATES",
											Value: "/var/run/cert-manager/certs/certs.pem",
										},
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          "metrics",
											ContainerPort: 10258,
											Protocol:      corev1.ProtocolTCP,
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/healthz",
												Port:   intstr.FromInt32(10258),
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
							Volumes: []corev1.Volume{
								{
									Name: "kubeconfig",
									VolumeSource: corev1.VolumeSource{
										Projected: &corev1.ProjectedVolumeSource{
											DefaultMode: ptr.To[int32](420),
											Sources: []corev1.VolumeProjection{
												{
													Secret: &corev1.SecretProjection{
														Items: []corev1.KeyToPath{
															{
																Key:  "kubeconfig",
																Path: "kubeconfig",
															},
														},
														LocalObjectReference: corev1.LocalObjectReference{Name: "generic-token-kubeconfig-71a3f1a4"},
														Optional:             ptr.To(false),
													},
												},
												{
													Secret: &corev1.SecretProjection{
														Items: []corev1.KeyToPath{
															{
																Key:  "token",
																Path: "token",
															},
														},
														LocalObjectReference: corev1.LocalObjectReference{Name: "shoot-access-extension-shoot-cert-service"},
														Optional:             ptr.To(false),
													},
												},
											},
										},
									},
								},
								{
									Name: "ca-certificates",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: "cert-controller-manager-ca-certificates"},
										},
									},
								},
							},
							ServiceAccountName: name,
						},
					},
				},
			}
			if !internal {
				obj.Labels["high-availability-config.resources.gardener.cloud/type"] = "controller"
			}
			if internal {
				container := &obj.Spec.Template.Spec.Containers[0]
				container.Name = "cert-management-" + certClass
				container.VolumeMounts = container.VolumeMounts[1:]
				obj.Spec.Template.Spec.Volumes = obj.Spec.Template.Spec.Volumes[1:]
			}
			return obj
		}

		standardInternalResources = func(namespace, certClass string) []client.Object {
			instance := "cert-management-" + certClass

			return []client.Object{
				&apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "certificates.cert.gardener.cloud",
					},
				},
				&apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "certificaterevocations.cert.gardener.cloud",
					},
				},
				&apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "issuers.cert.gardener.cloud",
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "extensions.gardener.cloud:extension-shoot-cert-service:" + certClass,
						Labels: map[string]string{
							"app.kubernetes.io/instance": instance,
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"networking.k8s.io"},
							Resources: []string{"ingresses"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{"gateway.networking.k8s.io"},
							Resources: []string{"gateways", "httproutes"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{"networking.istio.io"},
							Resources: []string{"gateways", "virtualservices"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"services"},
							Verbs:     []string{"get", "list", "update", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"cert.gardener.cloud"},
							Resources: []string{"certificates", "certificates/status", "certificaterevocations", "certificaterevocations/status"},
							Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"events"},
							Verbs:     []string{"create", "patch"},
						},
						{
							APIGroups: []string{"apiextensions.k8s.io"},
							Resources: []string{"customresourcedefinitions"},
							Verbs:     []string{"get", "list", "update", "create", "watch"},
						},
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "extensions.gardener.cloud:extension-shoot-cert-service:" + certClass,
						Labels: map[string]string{
							"app.kubernetes.io/instance": instance,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service:" + certClass,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      instance,
							Namespace: namespace,
						},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
						Namespace: namespace,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"configmaps"},
							Verbs:     []string{"create"},
						},
						{
							APIGroups:     []string{""},
							Resources:     []string{"configmaps"},
							Verbs:         []string{"get", "watch", "update"},
							ResourceNames: []string{"shoot-cert-service"},
						},
						{
							APIGroups: []string{"coordination.k8s.io"},
							Resources: []string{"leases"},
							Verbs:     []string{"create"},
						},
						{
							APIGroups:     []string{"coordination.k8s.io"},
							Resources:     []string{"leases"},
							ResourceNames: []string{"shoot-cert-service"},
							Verbs:         []string{"get", "watch", "update"},
						},
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
						Namespace: namespace,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      instance,
							Namespace: namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service:cert-controller-manager",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance,
						Namespace: namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
					},
					AutomountServiceAccountToken: ptr.To(false),
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager-ca-certificates",
						Namespace: namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
					},
					Data: map[string]string{
						"certs.pem": "cert1\ncert2\n",
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service",
						Namespace: namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
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
						{
							APIGroups: []string{"extensions.gardener.cloud"},
							Resources: []string{"dnsrecords"},
							Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"cert.gardener.cloud"},
							Resources: []string{"issuers", "issuers/status"},
							Verbs:     []string{"get", "update", "list", "patch", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"", "events.k8s.io"},
							Resources: []string{"events"},
							Verbs:     []string{"create", "patch"},
						},
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance,
						Namespace: namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      instance,
							Namespace: namespace,
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager",
						Namespace: namespace,
						Annotations: map[string]string{
							"networking.resources.gardener.cloud/from-all-scrape-targets-allowed-ports": "[{\"port\":10258,\"protocol\":\"TCP\"}]",
						},
						Labels: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
					},
					Spec: corev1.ServiceSpec{
						Type:      corev1.ServiceTypeClusterIP,
						ClusterIP: "None",
						Ports: []corev1.ServicePort{
							{
								Name:     "metrics",
								Port:     10258,
								Protocol: corev1.ProtocolTCP,
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/name":     instance,
							"app.kubernetes.io/instance": instance,
						},
					},
				},
				deployment(namespace, certClass, true),
			}
		}

		standardSeedResources = func() []client.Object {
			return []client.Object{
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						MaxUnavailable: &intstr.IntOrString{IntVal: 1},
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/name":     "shoot-cert-management-seed",
								"app.kubernetes.io/instance": "shoot-cert-management-seed",
							},
						},
						UnhealthyPodEvictionPolicy: ptr.To(policyv1.AlwaysAllow),
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shoot-cert-management-seed",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
					AutomountServiceAccountToken: ptr.To(false),
				},
				&certv1alpha1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "garden",
						Namespace: "shoot--foo--bar",
					},
					Spec: certv1alpha1.IssuerSpec{
						ACME: &certv1alpha1.ACMESpec{
							Server: "https://acme-v02.api.letsencrypt.org/directory",
							Email:  "foo@example.com",
							PrivateKeySecretRef: &corev1.SecretReference{
								Name:      "extension-shoot-cert-service-issuer-garden",
								Namespace: "shoot--foo--bar",
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extension-shoot-cert-service-issuer-garden",
						Namespace: "shoot--foo--bar",
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"email":      []byte("foo@example.com"),
						"privateKey": []byte("<private-key>"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager-ca-certificates",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
					Data: map[string]string{
						"certs.pem": "cert1\ncert2\n",
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extensions.gardener.cloud:extension-shoot-cert-service",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
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
						{
							APIGroups: []string{"dns.gardener.cloud"},
							Resources: []string{"dnsentries"},
							Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"cert.gardener.cloud"},
							Resources: []string{"issuers", "issuers/status"},
							Verbs:     []string{"get", "update", "list", "patch", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get", "list", "update", "patch", "watch", "create", "delete"},
						},
						{
							APIGroups: []string{"", "events.k8s.io"},
							Resources: []string{"events"},
							Verbs:     []string{"create", "patch"},
						},
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shoot-cert-management-seed",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "extensions.gardener.cloud:extension-shoot-cert-service",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "shoot-cert-management-seed",
							Namespace: "shoot--foo--bar",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager",
						Namespace: "shoot--foo--bar",
						Annotations: map[string]string{
							"networking.resources.gardener.cloud/from-all-scrape-targets-allowed-ports": "[{\"port\":10258,\"protocol\":\"TCP\"}]",
						},
						Labels: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
					Spec: corev1.ServiceSpec{
						Type:      corev1.ServiceTypeClusterIP,
						ClusterIP: "None",
						Ports: []corev1.ServicePort{
							{
								Name:     "metrics",
								Port:     10258,
								Protocol: corev1.ProtocolTCP,
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
						},
					},
				},
				deployment("shoot--foo--bar", "", false),
				&vpaautoscalingv1.VerticalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager-vpa",
						Namespace: "shoot--foo--bar",
					},
					Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
						TargetRef: &autoscalingv1.CrossVersionObjectReference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "cert-controller-manager",
						},
						UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
							UpdateMode: ptr.To(vpaautoscalingv1.UpdateModeAuto),
						},
						ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
							ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
								{
									ContainerName:    "*",
									ControlledValues: ptr.To(vpaautoscalingv1.ContainerControlledValuesRequestsOnly),
									ControlledResources: ptr.To([]corev1.ResourceName{
										corev1.ResourceMemory,
									}),
									MinAllowed: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("20Mi"),
									},
								},
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cert-controller-manager-dashboards",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"dashboard.monitoring.gardener.cloud/shoot": "true",
						},
					},
					Data: map[string]string{
						"cert-controller-manager-dashboard.json": "<autofilled>",
					},
				},
				&monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shoot-cert-controller-manager",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"prometheus":                 "shoot",
						},
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "cert-controller-manager.rules",
								Rules: []monitoringv1.Rule{
									{
										Alert: "SslCertificateWillExpireSoon",
										Expr:  intstr.FromString("((cert_management_cert_object_expire > 0) - time()) / 86400 <= 15"),
										For:   ptr.To[monitoringv1.Duration]("30m"),
										Labels: map[string]string{
											"service":    "cert-controller-manager",
											"severity":   "critical",
											"type":       "seed",
											"visibility": "operator",
										},
										Annotations: map[string]string{
											"description": "Certificate in namespace shoot--foo--bar will expire in less than 15 days.",
											"summary":     "TLS certificate will expire in less than 15 days",
										},
									},
								},
							},
						},
					},
				},
				&monitoringv1.ServiceMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shoot-cert-controller-manager",
						Namespace: "shoot--foo--bar",
						Labels: map[string]string{
							"app.kubernetes.io/instance": "shoot-cert-management-seed",
							"app.kubernetes.io/name":     "shoot-cert-management-seed",
							"prometheus":                 "shoot",
						},
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/name":     "shoot-cert-management-seed",
								"app.kubernetes.io/instance": "shoot-cert-management-seed",
							},
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
								HonorLabels: false,
							},
						},
					},
				},
			}
		}

		testSeedManagedResource = func(resources []client.Object, modifyDeployment func(*appsv1.Deployment)) {
			deployer := newDeployer(values)
			ExpectWithOffset(1, deployer.DeploySeedManagedResource(ctx, c)).To(Succeed())
			mr := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      servicev1alpha1.CertManagementResourceNameSeed,
					Namespace: values.Namespace,
				},
			}
			ExpectWithOffset(1, c.Get(ctx, client.ObjectKeyFromObject(mr), mr)).To(Succeed())
			ExpectWithOffset(1, mr.Spec.SecretRefs).To(HaveLen(1))
			completeObservabilityConfigMap(resources)
			if modifyDeployment != nil {
				for _, obj := range resources {
					if deployment, ok := obj.(*appsv1.Deployment); ok {
						modifyDeployment(deployment)
					}
				}
			}
			ExpectWithOffset(1, mr).To(consistOf(resources...))

			ExpectWithOffset(1, deployer.DeleteSeedManagedResourceAndWait(ctx, c, 5*time.Second))
			ExpectWithOffset(1, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(mr), mr))).To(BeTrue())
		}

		testInternalManagedResource = func(resources []client.Object, isGarden bool, modifyDeployment func(*appsv1.Deployment)) {
			deployer := newDeployer(values)
			ExpectWithOffset(1, deployer.DeployGardenOrSeedManagedResource(ctx, c)).To(Succeed())
			name := servicev1alpha1.CertManagementResourceNameSeed
			if isGarden {
				name = servicev1alpha1.CertManagementResourceNameGarden
			}
			mr := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: values.Namespace,
				},
			}
			ExpectWithOffset(1, c.Get(ctx, client.ObjectKeyFromObject(mr), mr)).To(Succeed())
			ExpectWithOffset(1, mr.Spec.SecretRefs).To(HaveLen(1))
			if modifyDeployment != nil {
				for _, obj := range resources {
					if deployment, ok := obj.(*appsv1.Deployment); ok {
						modifyDeployment(deployment)
					}
				}
			}
			crdCount, err := completeCRDs(resources, true)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, crdCount).To(Equal(3))

			ExpectWithOffset(1, mr).To(consistOf(resources...))

			ExpectWithOffset(1, deployer.DeleteGardenOrSeedManagedResourceAndWait(ctx, c, 5*time.Second))
			ExpectWithOffset(1, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(mr), mr))).To(BeTrue())
		}

		testShootManagedResource = func(resources []client.Object, expectedIssuerCRD bool) {
			deployer := newDeployer(values)
			ExpectWithOffset(1, deployer.DeployShootManagedResource(ctx, c)).To(Succeed())
			mr := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      servicev1alpha1.CertManagementResourceNameShoot,
					Namespace: values.Namespace,
				},
			}
			ExpectWithOffset(1, c.Get(ctx, client.ObjectKeyFromObject(mr), mr)).To(Succeed())
			ExpectWithOffset(1, mr.Spec.SecretRefs).To(HaveLen(1))
			crdCount, err := completeCRDs(resources, false)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			expectedCount := 2
			if expectedIssuerCRD {
				expectedCount++
			}
			ExpectWithOffset(1, crdCount).To(Equal(expectedCount))
			ExpectWithOffset(1, mr).To(consistOf(resources...))

			ExpectWithOffset(1, deployer.DeleteShootManagedResourceAndWait(ctx, c, 5*time.Second))
			ExpectWithOffset(1, errors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(mr), mr))).To(BeTrue())
		}
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(certserviceclient.ClusterScheme).Build()
		consistOf = matchers.NewManagedResourceConsistOfObjectsMatcher(c)
		values = Values{
			Namespace: "shoot--foo--bar",
			ExtensionConfig: config.Configuration{
				IssuerName:                 "garden",
				DefaultRequestsPerDayQuota: ptr.To[int32](100),
				ACME: &config.ACME{
					Email:                    "foo@example.com",
					Server:                   "https://acme-v02.api.letsencrypt.org/directory",
					PrivateKey:               ptr.To("<private-key>"),
					CACertificates:           ptr.To("cert1\ncert2\n"),
					DeactivateAuthorizations: ptr.To(true),
				},
			},
			ShootDeployment: true,
			Replicas:        1,

			Image:                            "example.com/gardener-project/releases/cert-controller-manager:v0.0.0",
			GenericTokenKubeconfigSecretName: "generic-token-kubeconfig-71a3f1a4",
		}
	})

	Describe("DeployShootManagedResource", func() {
		It("should deploy the standard shoot managed resource", func() {
			testShootManagedResource(standardShootResources(), false)
		})

		It("should deploy the shoot managed resource with DNS challenges on shoot", func() {
			values.CertConfig.DNSChallengeOnShoot = &service.DNSChallengeOnShoot{
				Enabled: true,
			}
			resources := standardShootResources()
			role := resources[2].(*rbacv1.ClusterRole)
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				APIGroups: []string{"dns.gardener.cloud"},
				Resources: []string{"dnsentries"},
				Verbs:     []string{"get", "list", "update", "watch", "create", "delete"},
			})
			testShootManagedResource(resources, false)
		})

		It("should deploy the shoot managed resource with issuers on shoot", func() {
			values.CertConfig.ShootIssuers = &service.ShootIssuers{
				Enabled: true,
			}
			resources := append(standardShootResources(), &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "issuers.cert.gardener.cloud",
				},
			})
			role := resources[2].(*rbacv1.ClusterRole)
			for i := range role.Rules {
				rule := &role.Rules[i]
				if rule.APIGroups[0] == "cert.gardener.cloud" {
					rule.Resources = append(rule.Resources, "issuers", "issuers/status")
				}
			}
			testShootManagedResource(resources, true)
		})
	})

	Describe("DeploySeedManagedResource", func() {
		It("should deploy it", func() {
			testSeedManagedResource(standardSeedResources(), nil)
		})

		It("should deploy with no replicas on hibernation", func() {
			values.Replicas = 0
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Replicas = ptr.To[int32](0)
			})
		})

		It("should deploy it with restricted default issuer", func() {
			values.ExtensionConfig.RestrictIssuer = ptr.To(true)
			values.RestrictedDomains = "sub1.example.com,sub2.example.com"
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"--issuer.default-requests-per-day-quota=",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--issuer.default-issuer-domain-ranges=sub1.example.com,sub2.example.com",
				)
			})
		})

		It("should deploy it with DNS challenges on shoot", func() {
			values.CertConfig.DNSChallengeOnShoot = &service.DNSChallengeOnShoot{
				Enabled:   true,
				DNSClass:  ptr.To("my-dns-class"),
				Namespace: "my-ns",
			}
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"--issuer.default-requests-per-day-quota=",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--dns=/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig",
					"--issuer.dns-namespace=my-ns",
					"--issuer.dns-class=my-dns-class",
				)
				deployment.Spec.Template.Spec.Containers[0].Args = removeArgs(deployment.Spec.Template.Spec.Containers[0].Args, "--issuer.dns-namespace=shoot--foo--bar")
			})
		})

		It("should deploy it with precheck nameservers", func() {
			values.ExtensionConfig.ACME.PrecheckNameservers = ptr.To("8.8.8.8,8.8.4.4")
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"--issuer.default-requests-per-day-quota=",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--issuer.precheck-nameservers=8.8.8.8,8.8.4.4",
				)
			})
		})

		It("should deploy it with issuers on shoot", func() {
			values.CertConfig.ShootIssuers = &service.ShootIssuers{
				Enabled: true,
			}
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"--issuer.default.pool.size=",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--allow-target-issuers",
				)
			})
		})

		It("should deploy it resource without alerting", func() {
			values.CertConfig.Alerting = &service.Alerting{CertExpirationAlertDays: ptr.To(0)}
			resources := excludeResourcesByType(standardSeedResources(), &monitoringv1.PrometheusRule{})
			testSeedManagedResource(resources, nil)
		})

		It("should deploy it resource with overwritten private key defaults", func() {
			values.ExtensionConfig.PrivateKeyDefaults = &config.PrivateKeyDefaults{
				Algorithm: ptr.To("ECDSA"),
				SizeRSA:   ptr.To(2048),
				SizeECDSA: ptr.To(256),
			}
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = removeArgs(deployment.Spec.Template.Spec.Containers[0].Args,
					"--default-private-key-algorithm=RSA",
					"--default-rsa-private-key-size=3072",
					"--default-ecdsa-private-key-size=384",
				)
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"<end>",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--default-private-key-algorithm=ECDSA",
					"--default-rsa-private-key-size=2048",
					"--default-ecdsa-private-key-size=256",
				)
			})
		})

		It("should deploy it with propagation timeout", func() {
			values.ExtensionConfig.ACME.PropagationTimeout = &metav1.Duration{Duration: 5 * time.Minute}
			testSeedManagedResource(standardSeedResources(), func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Args = insertArgsAfter(
					"--issuer.default.pool.size=",
					deployment.Spec.Template.Spec.Containers[0].Args,
					"--propagation-timeout=5m0s",
				)
			})
		})

		It("should deploy it with additional issuers", func() {
			values.CertConfig.Issuers = []service.IssuerConfig{
				{
					Name:                 "bar",
					Server:               "https://acme.example.com/directory",
					Email:                "bar@example.com",
					RequestsPerDayQuota:  ptr.To(999),
					PrivateKeySecretName: ptr.To("bar-secret"),
					ExternalAccountBinding: &service.ACMEExternalAccountBinding{
						KeyID:         "key-id",
						KeySecretName: "eab-secret",
					},
					SkipDNSChallengeValidation: ptr.To(true),
					Domains: &service.DNSSelection{
						Include: []string{"example.com"},
						Exclude: []string{"sub.example.com"},
					},
					PrecheckNameservers: []string{"1.1.1.1", "2.2.2.2"},
				},
				{
					Name:   "bar2",
					Server: "https://acme2.example.com/directory",
					Email:  "bar2@example.com",
				},
			}
			values.Resources = []gardencorev1beta1.NamedResourceReference{
				{Name: "bar-secret", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "original-bar-secret", Kind: "Secret"}},
				{Name: "eab-secret", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "original-bar-secret", Kind: "Secret"}},
			}
			resources := append(standardSeedResources(),
				&certv1alpha1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "shoot--foo--bar",
					},
					Spec: certv1alpha1.IssuerSpec{
						ACME: &certv1alpha1.ACMESpec{
							Server:           "https://acme.example.com/directory",
							Email:            "bar@example.com",
							AutoRegistration: false,
							PrivateKeySecretRef: &corev1.SecretReference{
								Name:      "ref-original-bar-secret",
								Namespace: "shoot--foo--bar",
							},
							ExternalAccountBinding: &certv1alpha1.ACMEExternalAccountBinding{
								KeyID: "key-id",
								KeySecretRef: &corev1.SecretReference{
									Name:      "ref-original-bar-secret",
									Namespace: "shoot--foo--bar",
								},
							},
							SkipDNSChallengeValidation: ptr.To(true),
							Domains: &certv1alpha1.DNSSelection{
								Include: []string{"example.com"},
								Exclude: []string{"sub.example.com"},
							},
							PrecheckNameservers: []string{"1.1.1.1", "2.2.2.2"},
						},
						RequestsPerDayQuota: ptr.To(999),
					},
				},
				&certv1alpha1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar2",
						Namespace: "shoot--foo--bar",
					},
					Spec: certv1alpha1.IssuerSpec{
						ACME: &certv1alpha1.ACMESpec{
							Server:           "https://acme2.example.com/directory",
							Email:            "bar2@example.com",
							AutoRegistration: true,
							PrivateKeySecretRef: &corev1.SecretReference{
								Name:      "extension-shoot-cert-service-issuer-bar2",
								Namespace: "shoot--foo--bar",
							},
						},
					},
				},
			)
			testSeedManagedResource(resources, func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Annotations = map[string]string{"checksum/issuers": "51971e0765b20445e8208abb457f19126570c241cc303c6a37faf82d3f6d79b4"}
			})
		})
	})

	Describe("DeployGardenOrSeedManagedResource", func() {
		It("should deploy it for the runtime cluster with self-signed root CA", func() {
			values.ShootDeployment = false
			values.Namespace = "garden"
			values.CertClass = "garden"
			values.ExtensionConfig.CA = &config.CA{
				Certificate:    "certificate",
				CertificateKey: "cert-key",
				CACertificates: values.ExtensionConfig.ACME.CACertificates,
			}
			values.ExtensionConfig.ACME = nil
			resources := standardInternalResources(values.Namespace, values.CertClass)
			resources = append(resources,
				&certv1alpha1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "garden",
						Namespace: values.Namespace,
					},
					Spec: certv1alpha1.IssuerSpec{
						CA: &certv1alpha1.CASpec{
							PrivateKeySecretRef: &corev1.SecretReference{
								Name:      "extension-shoot-cert-service-issuer-garden-ca",
								Namespace: values.Namespace,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extension-shoot-cert-service-issuer-garden-ca",
						Namespace: values.Namespace,
					},
					Type: corev1.SecretTypeTLS,
					Data: map[string][]byte{
						"tls.crt": []byte("certificate"),
						"tls.key": []byte("cert-key"),
					},
				})
			testInternalManagedResource(resources, true, func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Annotations = map[string]string{
					"checksum/issuers": "4eb1941a6e4f4d326b96278513c50d462df6c24d9566c0f126d34f40eeb6a506",
				}
				deployment.Spec.Template.Spec.Containers[0].Args = removeArgs(deployment.Spec.Template.Spec.Containers[0].Args, "--acme-deactivate-authorizations")
			})
		})

		It("should deploy it for a seed cluster", func() {
			values.ShootDeployment = false
			values.Namespace = "seed-foo"
			values.CertClass = "seed"
			resources := standardInternalResources(values.Namespace, values.CertClass)
			resources = append(resources,
				&certv1alpha1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "garden",
						Namespace: values.Namespace,
					},
					Spec: certv1alpha1.IssuerSpec{
						ACME: &certv1alpha1.ACMESpec{
							Server: "https://acme-v02.api.letsencrypt.org/directory",
							Email:  "foo@example.com",
							PrivateKeySecretRef: &corev1.SecretReference{
								Name:      "extension-shoot-cert-service-issuer-garden",
								Namespace: values.Namespace,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extension-shoot-cert-service-issuer-garden",
						Namespace: values.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"email":      []byte("foo@example.com"),
						"privateKey": []byte("<private-key>"),
					},
				})
			testInternalManagedResource(resources, false, nil)
		})
	})
})

func completeCRDs(objects []client.Object, keepObject bool) (int, error) {
	count := 0
	for i, obj := range objects {
		var data string
		switch obj.GetName() {
		case "certificates.cert.gardener.cloud":
			data = crdCertificates
		case "certificaterevocations.cert.gardener.cloud":
			data = crdRevocations
		case "issuers.cert.gardener.cloud":
			data = crdIssuers
		}
		if data == "" {
			continue
		}
		count++
		crd, err := stringToCRD(data)
		if err != nil {
			return i, err
		}
		if keepObject {
			crd.GetAnnotations()[resourcesv1alpha1.KeepObject] = "true"
		}
		crd.SetLabels(map[string]string{v1beta1constants.ShootNoCleanup: "true"})
		objects[i] = crd
	}
	return count, nil
}

func completeObservabilityConfigMap(objects []client.Object) {
	for _, obj := range objects {
		if cm, ok := obj.(*corev1.ConfigMap); ok && cm.Name == "cert-controller-manager-observability-config" {
			cm.Data["dashboard_operators"] = certDashboardJSON
			cm.Data["dashboard_users"] = certDashboardJSON
		}
		if cm, ok := obj.(*corev1.ConfigMap); ok && cm.Name == "cert-controller-manager-dashboards" {
			cm.Data["cert-controller-manager-dashboard.json"] = certDashboardJSON
		}
	}
}

func insertArgsAfter(afterArgPrefix string, args []string, insertArgs ...string) []string {
	for i, arg := range args {
		if strings.HasPrefix(arg, afterArgPrefix) {
			return append(args[:i+1], append(insertArgs, args[i+1:]...)...)
		}
	}
	return append(args, insertArgs...)
}

func removeArgs(args []string, removeArgs ...string) []string {
	var newArgs []string
	for _, arg := range args {
		remove := false
		for _, removeArg := range removeArgs {
			if arg == removeArg {
				remove = true
				break
			}
		}
		if !remove {
			newArgs = append(newArgs, arg)
		}
	}
	return newArgs
}

func excludeResourcesByType(resources []client.Object, excludes ...client.Object) []client.Object {
	var filtered []client.Object
	for _, obj := range resources {
		excluded := false
		for _, other := range excludes {
			if reflect.TypeOf(obj) == reflect.TypeOf(other) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}
