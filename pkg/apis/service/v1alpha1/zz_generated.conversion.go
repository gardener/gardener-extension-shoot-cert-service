//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	service "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/service"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*ACMEExternalAccountBinding)(nil), (*service.ACMEExternalAccountBinding)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ACMEExternalAccountBinding_To_service_ACMEExternalAccountBinding(a.(*ACMEExternalAccountBinding), b.(*service.ACMEExternalAccountBinding), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.ACMEExternalAccountBinding)(nil), (*ACMEExternalAccountBinding)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_ACMEExternalAccountBinding_To_v1alpha1_ACMEExternalAccountBinding(a.(*service.ACMEExternalAccountBinding), b.(*ACMEExternalAccountBinding), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*CertConfig)(nil), (*service.CertConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CertConfig_To_service_CertConfig(a.(*CertConfig), b.(*service.CertConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.CertConfig)(nil), (*CertConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_CertConfig_To_v1alpha1_CertConfig(a.(*service.CertConfig), b.(*CertConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*DNSChallengeOnShoot)(nil), (*service.DNSChallengeOnShoot)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_DNSChallengeOnShoot_To_service_DNSChallengeOnShoot(a.(*DNSChallengeOnShoot), b.(*service.DNSChallengeOnShoot), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.DNSChallengeOnShoot)(nil), (*DNSChallengeOnShoot)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_DNSChallengeOnShoot_To_v1alpha1_DNSChallengeOnShoot(a.(*service.DNSChallengeOnShoot), b.(*DNSChallengeOnShoot), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*DNSSelection)(nil), (*service.DNSSelection)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_DNSSelection_To_service_DNSSelection(a.(*DNSSelection), b.(*service.DNSSelection), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.DNSSelection)(nil), (*DNSSelection)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_DNSSelection_To_v1alpha1_DNSSelection(a.(*service.DNSSelection), b.(*DNSSelection), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*IssuerConfig)(nil), (*service.IssuerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_IssuerConfig_To_service_IssuerConfig(a.(*IssuerConfig), b.(*service.IssuerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.IssuerConfig)(nil), (*IssuerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_IssuerConfig_To_v1alpha1_IssuerConfig(a.(*service.IssuerConfig), b.(*IssuerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ShootIssuers)(nil), (*service.ShootIssuers)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ShootIssuers_To_service_ShootIssuers(a.(*ShootIssuers), b.(*service.ShootIssuers), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*service.ShootIssuers)(nil), (*ShootIssuers)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_service_ShootIssuers_To_v1alpha1_ShootIssuers(a.(*service.ShootIssuers), b.(*ShootIssuers), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_ACMEExternalAccountBinding_To_service_ACMEExternalAccountBinding(in *ACMEExternalAccountBinding, out *service.ACMEExternalAccountBinding, s conversion.Scope) error {
	out.KeyID = in.KeyID
	out.KeySecretName = in.KeySecretName
	return nil
}

// Convert_v1alpha1_ACMEExternalAccountBinding_To_service_ACMEExternalAccountBinding is an autogenerated conversion function.
func Convert_v1alpha1_ACMEExternalAccountBinding_To_service_ACMEExternalAccountBinding(in *ACMEExternalAccountBinding, out *service.ACMEExternalAccountBinding, s conversion.Scope) error {
	return autoConvert_v1alpha1_ACMEExternalAccountBinding_To_service_ACMEExternalAccountBinding(in, out, s)
}

func autoConvert_service_ACMEExternalAccountBinding_To_v1alpha1_ACMEExternalAccountBinding(in *service.ACMEExternalAccountBinding, out *ACMEExternalAccountBinding, s conversion.Scope) error {
	out.KeyID = in.KeyID
	out.KeySecretName = in.KeySecretName
	return nil
}

// Convert_service_ACMEExternalAccountBinding_To_v1alpha1_ACMEExternalAccountBinding is an autogenerated conversion function.
func Convert_service_ACMEExternalAccountBinding_To_v1alpha1_ACMEExternalAccountBinding(in *service.ACMEExternalAccountBinding, out *ACMEExternalAccountBinding, s conversion.Scope) error {
	return autoConvert_service_ACMEExternalAccountBinding_To_v1alpha1_ACMEExternalAccountBinding(in, out, s)
}

func autoConvert_v1alpha1_CertConfig_To_service_CertConfig(in *CertConfig, out *service.CertConfig, s conversion.Scope) error {
	out.Issuers = *(*[]service.IssuerConfig)(unsafe.Pointer(&in.Issuers))
	out.DNSChallengeOnShoot = (*service.DNSChallengeOnShoot)(unsafe.Pointer(in.DNSChallengeOnShoot))
	out.ShootIssuers = (*service.ShootIssuers)(unsafe.Pointer(in.ShootIssuers))
	out.PrecheckNameservers = (*string)(unsafe.Pointer(in.PrecheckNameservers))
	return nil
}

// Convert_v1alpha1_CertConfig_To_service_CertConfig is an autogenerated conversion function.
func Convert_v1alpha1_CertConfig_To_service_CertConfig(in *CertConfig, out *service.CertConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_CertConfig_To_service_CertConfig(in, out, s)
}

func autoConvert_service_CertConfig_To_v1alpha1_CertConfig(in *service.CertConfig, out *CertConfig, s conversion.Scope) error {
	out.Issuers = *(*[]IssuerConfig)(unsafe.Pointer(&in.Issuers))
	out.DNSChallengeOnShoot = (*DNSChallengeOnShoot)(unsafe.Pointer(in.DNSChallengeOnShoot))
	out.ShootIssuers = (*ShootIssuers)(unsafe.Pointer(in.ShootIssuers))
	out.PrecheckNameservers = (*string)(unsafe.Pointer(in.PrecheckNameservers))
	return nil
}

// Convert_service_CertConfig_To_v1alpha1_CertConfig is an autogenerated conversion function.
func Convert_service_CertConfig_To_v1alpha1_CertConfig(in *service.CertConfig, out *CertConfig, s conversion.Scope) error {
	return autoConvert_service_CertConfig_To_v1alpha1_CertConfig(in, out, s)
}

func autoConvert_v1alpha1_DNSChallengeOnShoot_To_service_DNSChallengeOnShoot(in *DNSChallengeOnShoot, out *service.DNSChallengeOnShoot, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.Namespace = in.Namespace
	out.DNSClass = (*string)(unsafe.Pointer(in.DNSClass))
	return nil
}

// Convert_v1alpha1_DNSChallengeOnShoot_To_service_DNSChallengeOnShoot is an autogenerated conversion function.
func Convert_v1alpha1_DNSChallengeOnShoot_To_service_DNSChallengeOnShoot(in *DNSChallengeOnShoot, out *service.DNSChallengeOnShoot, s conversion.Scope) error {
	return autoConvert_v1alpha1_DNSChallengeOnShoot_To_service_DNSChallengeOnShoot(in, out, s)
}

func autoConvert_service_DNSChallengeOnShoot_To_v1alpha1_DNSChallengeOnShoot(in *service.DNSChallengeOnShoot, out *DNSChallengeOnShoot, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.Namespace = in.Namespace
	out.DNSClass = (*string)(unsafe.Pointer(in.DNSClass))
	return nil
}

// Convert_service_DNSChallengeOnShoot_To_v1alpha1_DNSChallengeOnShoot is an autogenerated conversion function.
func Convert_service_DNSChallengeOnShoot_To_v1alpha1_DNSChallengeOnShoot(in *service.DNSChallengeOnShoot, out *DNSChallengeOnShoot, s conversion.Scope) error {
	return autoConvert_service_DNSChallengeOnShoot_To_v1alpha1_DNSChallengeOnShoot(in, out, s)
}

func autoConvert_v1alpha1_DNSSelection_To_service_DNSSelection(in *DNSSelection, out *service.DNSSelection, s conversion.Scope) error {
	out.Include = *(*[]string)(unsafe.Pointer(&in.Include))
	out.Exclude = *(*[]string)(unsafe.Pointer(&in.Exclude))
	return nil
}

// Convert_v1alpha1_DNSSelection_To_service_DNSSelection is an autogenerated conversion function.
func Convert_v1alpha1_DNSSelection_To_service_DNSSelection(in *DNSSelection, out *service.DNSSelection, s conversion.Scope) error {
	return autoConvert_v1alpha1_DNSSelection_To_service_DNSSelection(in, out, s)
}

func autoConvert_service_DNSSelection_To_v1alpha1_DNSSelection(in *service.DNSSelection, out *DNSSelection, s conversion.Scope) error {
	out.Include = *(*[]string)(unsafe.Pointer(&in.Include))
	out.Exclude = *(*[]string)(unsafe.Pointer(&in.Exclude))
	return nil
}

// Convert_service_DNSSelection_To_v1alpha1_DNSSelection is an autogenerated conversion function.
func Convert_service_DNSSelection_To_v1alpha1_DNSSelection(in *service.DNSSelection, out *DNSSelection, s conversion.Scope) error {
	return autoConvert_service_DNSSelection_To_v1alpha1_DNSSelection(in, out, s)
}

func autoConvert_v1alpha1_IssuerConfig_To_service_IssuerConfig(in *IssuerConfig, out *service.IssuerConfig, s conversion.Scope) error {
	out.Name = in.Name
	out.Server = in.Server
	out.Email = in.Email
	out.RequestsPerDayQuota = (*int)(unsafe.Pointer(in.RequestsPerDayQuota))
	out.PrivateKeySecretName = (*string)(unsafe.Pointer(in.PrivateKeySecretName))
	out.ExternalAccountBinding = (*service.ACMEExternalAccountBinding)(unsafe.Pointer(in.ExternalAccountBinding))
	out.SkipDNSChallengeValidation = (*bool)(unsafe.Pointer(in.SkipDNSChallengeValidation))
	out.Domains = (*service.DNSSelection)(unsafe.Pointer(in.Domains))
	out.PrecheckNameservers = *(*[]string)(unsafe.Pointer(&in.PrecheckNameservers))
	return nil
}

// Convert_v1alpha1_IssuerConfig_To_service_IssuerConfig is an autogenerated conversion function.
func Convert_v1alpha1_IssuerConfig_To_service_IssuerConfig(in *IssuerConfig, out *service.IssuerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_IssuerConfig_To_service_IssuerConfig(in, out, s)
}

func autoConvert_service_IssuerConfig_To_v1alpha1_IssuerConfig(in *service.IssuerConfig, out *IssuerConfig, s conversion.Scope) error {
	out.Name = in.Name
	out.Server = in.Server
	out.Email = in.Email
	out.RequestsPerDayQuota = (*int)(unsafe.Pointer(in.RequestsPerDayQuota))
	out.PrivateKeySecretName = (*string)(unsafe.Pointer(in.PrivateKeySecretName))
	out.ExternalAccountBinding = (*ACMEExternalAccountBinding)(unsafe.Pointer(in.ExternalAccountBinding))
	out.SkipDNSChallengeValidation = (*bool)(unsafe.Pointer(in.SkipDNSChallengeValidation))
	out.Domains = (*DNSSelection)(unsafe.Pointer(in.Domains))
	out.PrecheckNameservers = *(*[]string)(unsafe.Pointer(&in.PrecheckNameservers))
	return nil
}

// Convert_service_IssuerConfig_To_v1alpha1_IssuerConfig is an autogenerated conversion function.
func Convert_service_IssuerConfig_To_v1alpha1_IssuerConfig(in *service.IssuerConfig, out *IssuerConfig, s conversion.Scope) error {
	return autoConvert_service_IssuerConfig_To_v1alpha1_IssuerConfig(in, out, s)
}

func autoConvert_v1alpha1_ShootIssuers_To_service_ShootIssuers(in *ShootIssuers, out *service.ShootIssuers, s conversion.Scope) error {
	out.Enabled = in.Enabled
	return nil
}

// Convert_v1alpha1_ShootIssuers_To_service_ShootIssuers is an autogenerated conversion function.
func Convert_v1alpha1_ShootIssuers_To_service_ShootIssuers(in *ShootIssuers, out *service.ShootIssuers, s conversion.Scope) error {
	return autoConvert_v1alpha1_ShootIssuers_To_service_ShootIssuers(in, out, s)
}

func autoConvert_service_ShootIssuers_To_v1alpha1_ShootIssuers(in *service.ShootIssuers, out *ShootIssuers, s conversion.Scope) error {
	out.Enabled = in.Enabled
	return nil
}

// Convert_service_ShootIssuers_To_v1alpha1_ShootIssuers is an autogenerated conversion function.
func Convert_service_ShootIssuers_To_v1alpha1_ShootIssuers(in *service.ShootIssuers, out *ShootIssuers, s conversion.Scope) error {
	return autoConvert_service_ShootIssuers_To_v1alpha1_ShootIssuers(in, out, s)
}
