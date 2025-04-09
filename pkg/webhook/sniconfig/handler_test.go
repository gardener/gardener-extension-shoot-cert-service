// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package sniconfig_test

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	gomegatypes "github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
	. "github.com/gardener/gardener-extension-shoot-cert-service/pkg/webhook/sniconfig"
)

var _ = Describe("handler", func() {
	Describe("#Handle", func() {
		var (
			ctx = context.TODO()
			log logr.Logger

			request admission.Request
			decoder admission.Decoder
			handler admission.Handler

			ctrl *gomock.Controller
			c    *mockclient.MockClient

			fooKind         = metav1.GroupVersionKind{Group: "foo", Version: "bar", Kind: "Foo"}
			deployment      *appsv1.Deployment
			expectedPatches []jsonpatch.JsonPatchOperation
		)

		BeforeEach(func() {
			format.MaxLength = 4000
			os.Setenv("VIRTUAL_KUBE_API_SERVER_SNI_INCLUDE_PRIMARY_DOMAIN", "true")
			log = logger.MustNewZapLogger(logger.DebugLevel, logger.FormatJSON, logzap.WriteTo(GinkgoWriter))

			ctrl = gomock.NewController(GinkgoT())
			c = mockclient.NewMockClient(ctrl)

			request = admission.Request{}
			request.Operation = admissionv1.Update
			request.Kind = metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}

			var err error
			decoder = admission.NewDecoder(kubernetes.SeedScheme)
			Expect(err).NotTo(HaveOccurred())

			handler = &Handler{Logger: log, TargetClient: c, Decoder: decoder}

			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virtual-garden-kube-apiserver",
					Namespace: "garden",
					Annotations: map[string]string{
						certificate.TLSCertAPIServerNamesAnnotation: "foo.example.com",
						certificate.TLSCertHashAnnotation:           "1234",
						certificate.TLSCertRequestedAtAnnotation:    "2000-01-01T00:00:00Z",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "kube-apiserver",
								Args: []string{
									"--foo=bar",
								},
							}},
						},
					},
				},
			}
			request.Name = deployment.Name
			request.Namespace = deployment.Namespace

			expectedPatches = []jsonpatch.JsonPatchOperation{
				{
					Operation: "add",
					Path:      "/spec/template/metadata/annotations",
					Value: map[string]any{
						certificate.TLSCertRequestedAtAnnotation: "2000-01-01T00:00:00Z",
						certificate.TLSCertHashAnnotation:        "1234",
					},
				},
				{
					Operation: "add",
					Path:      "/spec/template/spec/volumes",
					Value: []any{map[string]any{
						"name": "tls-sni-shoot-cert-service-injected",
						"secret": map[string]any{
							"secretName":  "tls",
							"defaultMode": json.Number("416"),
						},
					}},
				},
				{
					Operation: "add",
					Path:      "/spec/template/spec/containers/0/args/1",
					Value:     "--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:foo.example.com",
				},
				{
					Operation: "add",
					Path:      "/spec/template/spec/containers/0/volumeMounts",
					Value: []any{map[string]any{
						"name":      "tls-sni-shoot-cert-service-injected",
						"readOnly":  true,
						"mountPath": "/srv/kubernetes/tls-sni/shoot-cert-service-injected",
					}},
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("ignored requests", func() {
			It("should ignore other operations than CREATE or UPDATE", func() {
				for _, op := range []admissionv1.Operation{admissionv1.Delete, admissionv1.Connect} {
					request.Operation = op
					expectAllowed(handler.Handle(ctx, request), ContainSubstring("no changes required"))
				}
			})

			It("should ignore types other than deployment resources", func() {
				request.Kind = fooKind
				expectAllowed(handler.Handle(ctx, request), ContainSubstring("unexpected resource"))
			})

			It("should ignore other deployments than virtual-garden-kube-apiserver", func() {
				deployment.Name = "foo"
				prepareRequest(&request, deployment)
				expectAllowed(handler.Handle(ctx, request), ContainSubstring("no changes required"))
			})
		})

		Context("mutating requests", func() {
			It("should update the deployment on create operation", func() {
				prepareRequest(&request, deployment)
				request.Operation = admissionv1.Create
				expectPatched(handler.Handle(ctx, request), expectedPatches)
			})
			It("should update the deployment on update operation", func() {
				prepareRequest(&request, deployment)
				request.Operation = admissionv1.Update
				expectPatched(handler.Handle(ctx, request), expectedPatches)
			})
			It("should keep the deployment unchanged", func() {
				patchDeployment(deployment)
				prepareRequest(&request, deployment)
				expectPatched(handler.Handle(ctx, request), []jsonpatch.JsonPatchOperation{})
			})
			It("should drop additional args", func() {
				patchDeployment(deployment)
				deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args,
					"--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:bar.example.com",
					"--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:baz.example.com",
				)
				prepareRequest(&request, deployment)
				expectPatched(handler.Handle(ctx, request), []jsonpatch.JsonPatchOperation{
					{
						Operation: "remove",
						Path:      "/spec/template/spec/containers/0/args/2",
					},
					{
						Operation: "remove",
						Path:      "/spec/template/spec/containers/0/args/3",
					},
				})
			})
			It("should remove the injected stuff if deployment annotations are removed", func() {
				patchDeployment(deployment)
				deployment.Annotations = map[string]string{}
				prepareRequest(&request, deployment)
				expectPatched(handler.Handle(ctx, request), []jsonpatch.JsonPatchOperation{
					{
						Operation: "remove",
						Path:      "/spec/template/metadata/annotations",
					},
					{
						Operation: "remove",
						Path:      "/spec/template/spec/containers/0/args/1",
					},
					{
						Operation: "remove",
						Path:      "/spec/template/spec/containers/0/volumeMounts",
					},
					{
						Operation: "remove",
						Path:      "/spec/template/spec/volumes",
					},
				})
			})
		})
	})
})

func expectAllowed(response admission.Response, reason gomegatypes.GomegaMatcher, optionalDescription ...any) {
	ExpectWithOffset(1, response.Allowed).To(BeTrue(), optionalDescription...)
	ExpectWithOffset(1, response.Result.Message).To(reason, optionalDescription...)
}

func expectPatched(response admission.Response, expectedPatches []jsonpatch.JsonPatchOperation, optionalDescription ...any) {
	ExpectWithOffset(1, response.Allowed).To(BeTrue(), optionalDescription...)
	sort.Slice(response.Patches, func(i, j int) bool {
		return response.Patches[i].Path < response.Patches[j].Path
	})
	sort.Slice(expectedPatches, func(i, j int) bool {
		return expectedPatches[i].Path < expectedPatches[j].Path
	})
	ExpectWithOffset(1, response.Patches).To(Equal(expectedPatches), optionalDescription...)
}

func prepareRequest(request *admission.Request, obj *appsv1.Deployment) {
	objJSON, err := json.Marshal(obj)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	request.Object = runtime.RawExtension{Raw: objJSON}
	request.Name = obj.Name
	request.Namespace = obj.Namespace
}

func patchDeployment(deployment *appsv1.Deployment) {
	deployment.Spec.Template.Annotations = map[string]string{
		certificate.TLSCertHashAnnotation:        "1234",
		certificate.TLSCertRequestedAtAnnotation: "2000-01-01T00:00:00Z",
	}
	deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args,
		"--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:foo.example.com",
	)
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "tls-sni-shoot-cert-service-injected",
			MountPath: "/srv/kubernetes/tls-sni/shoot-cert-service-injected",
			ReadOnly:  true,
		})
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: "tls-sni-shoot-cert-service-injected",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "tls",
					DefaultMode: ptr.To(int32(416)),
				},
			},
		})
}
