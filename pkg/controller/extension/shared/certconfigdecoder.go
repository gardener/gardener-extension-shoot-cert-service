// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service/validation"
)

// CertConfigDecoder is responsible for decoding and validating the cert config.
type CertConfigDecoder struct {
	decoder runtime.Decoder
}

// NewCertConfigDecoder creates a new instance of CertConfigDecoder.
func NewCertConfigDecoder(mgr manager.Manager) CertConfigDecoder {
	return CertConfigDecoder{
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

// DecodeAndValidateProviderConfig decodes the provider config from the given Extension and validates it.
func (d *CertConfigDecoder) DecodeAndValidateProviderConfig(ex *extensionsv1alpha1.Extension, cluster *controller.Cluster) (*service.CertConfig, error) {
	certConfig := &service.CertConfig{}
	if ex.Spec.ProviderConfig != nil {
		if _, _, err := d.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, certConfig); err != nil {
			return nil, fmt.Errorf("failed to decode provider config: %+v", err)
		}
		if errs := validation.ValidateCertConfig(certConfig, cluster); len(errs) > 0 {
			return nil, errs.ToAggregate()
		}
	}
	return certConfig, nil
}
