// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package sniconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/controller/runtimecluster/certificate"
)

// Handler handles admission requests for deployment of virtual-garden-kube-apiserver and configures the SNI command line arguments.
type Handler struct {
	Logger       logr.Logger
	TargetClient client.Reader
	Decoder      admission.Decoder
}

// Handle defaults the high availability settings of the provided resource.
func (h *Handler) Handle(_ context.Context, req admission.Request) admission.Response {
	var (
		requestGK = schema.GroupKind{Group: req.Kind.Group, Kind: req.Kind.Kind}
		obj       runtime.Object
		err       error
	)

	switch requestGK {
	case appsv1.SchemeGroupVersion.WithKind("Deployment").GroupKind():
		obj, err = h.handleDeployment(req)
	default:
		return admission.Allowed(fmt.Sprintf("unexpected resource: %s", requestGK))
	}

	if err != nil {
		var apiStatus apierrors.APIStatus
		if errors.As(err, &apiStatus) {
			result := apiStatus.Status()
			return admission.Response{AdmissionResponse: admissionv1.AdmissionResponse{Allowed: false, Result: &result}}
		}
		return admission.Denied(err.Error())
	}
	if obj == nil {
		return admission.Allowed("no changes required")
	}

	marshalled, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshalled)
}

func (h *Handler) handleDeployment(req admission.Request) (runtime.Object, error) {
	if req.Name != "virtual-garden-kube-apiserver" || req.Namespace != "garden" {
		return nil, nil
	}
	if req.Operation != admissionv1.Update && req.Operation != admissionv1.Create {
		return nil, nil
	}

	deployment := &appsv1.Deployment{}
	if err := h.Decoder.Decode(req, deployment); err != nil {
		return nil, err
	}

	if err := mutateTLSCertSNI(h.Logger, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

func mutateTLSCertSNI(log logr.Logger, deployment *appsv1.Deployment) error {
	updateTemplateAnnotations(deployment)

	var apiServerNames []string
	if deployment.Annotations[certificate.TLSCertAPIServerNamesAnnotation] != "" {
		apiServerNames = strings.Split(deployment.Annotations[certificate.TLSCertAPIServerNamesAnnotation], ",")
	}
	if len(apiServerNames) > 0 && os.Getenv("VIRTUAL_KUBE_API_SERVER_SNI_INCLUDE_PRIMARY_DOMAIN") != "true" {
		apiServerNames = apiServerNames[1:]
	}
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if container.Name == "kube-apiserver" {
			// remove old args
			var oldAPIServerNames []string

			for i := len(container.Args) - 1; i >= 0; i-- {
				if strings.HasPrefix(container.Args[i], "--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/") {
					container.Args = append(container.Args[:i], container.Args[i+1:]...)
					// no break here, as there could be multiple args
				}
			}
			// add new args
			for _, apiServerName := range apiServerNames {
				container.Args = append(container.Args, fmt.Sprintf(
					"--tls-sni-cert-key=/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.crt,/srv/kubernetes/tls-sni/shoot-cert-service-injected/tls.key:%s",
					apiServerName))
			}
			if !reflect.DeepEqual(oldAPIServerNames, apiServerNames) {
				log.Info("updated tls-cert-sni domain names", "domainNames", strings.Join(apiServerNames, ","))
			}

			// remove old volume mount
			for i, volume := range container.VolumeMounts {
				if volume.Name == "tls-sni-shoot-cert-service-injected" {
					container.VolumeMounts = append(container.VolumeMounts[:i], container.VolumeMounts[i+1:]...)
					break
				}
			}
			// add new volume mount
			if len(apiServerNames) > 0 {
				container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
					Name:      "tls-sni-shoot-cert-service-injected",
					MountPath: "/srv/kubernetes/tls-sni/shoot-cert-service-injected",
					ReadOnly:  true,
				})
			}
		}
	}
	// remove old volume
	for i, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == "tls-sni-shoot-cert-service-injected" {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes[:i], deployment.Spec.Template.Spec.Volumes[i+1:]...)
			break
		}
	}
	// add new volume
	if len(apiServerNames) > 0 {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "tls-sni-shoot-cert-service-injected",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "tls",
					DefaultMode: ptr.To(int32(416)),
				},
			},
		})
	}

	return nil
}

func updateTemplateAnnotations(deployment *appsv1.Deployment) {
	certHash := deployment.Annotations[certificate.TLSCertHashAnnotation]
	certRequestedAt := deployment.Annotations[certificate.TLSCertRequestedAtAnnotation]

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}

	if certHash != "" {
		deployment.Spec.Template.Annotations[certificate.TLSCertHashAnnotation] = certHash
	} else {
		delete(deployment.Spec.Template.Annotations, certificate.TLSCertHashAnnotation)
	}
	if certRequestedAt != "" {
		deployment.Spec.Template.Annotations[certificate.TLSCertRequestedAtAnnotation] = certRequestedAt
	} else {
		delete(deployment.Spec.Template.Annotations, certificate.TLSCertRequestedAtAnnotation)
	}
}
