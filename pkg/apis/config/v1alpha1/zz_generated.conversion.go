// +build !ignore_autogenerated

/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

	config "github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
	healthcheckconfig "github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config"
	configv1alpha1 "github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config/v1alpha1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*ACME)(nil), (*config.ACME)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ACME_To_config_ACME(a.(*ACME), b.(*config.ACME), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.ACME)(nil), (*ACME)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_ACME_To_v1alpha1_ACME(a.(*config.ACME), b.(*ACME), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Configuration)(nil), (*config.Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Configuration_To_config_Configuration(a.(*Configuration), b.(*config.Configuration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.Configuration)(nil), (*Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_Configuration_To_v1alpha1_Configuration(a.(*config.Configuration), b.(*Configuration), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_ACME_To_config_ACME(in *ACME, out *config.ACME, s conversion.Scope) error {
	out.Email = in.Email
	out.Server = in.Server
	out.PrivateKey = (*string)(unsafe.Pointer(in.PrivateKey))
	return nil
}

// Convert_v1alpha1_ACME_To_config_ACME is an autogenerated conversion function.
func Convert_v1alpha1_ACME_To_config_ACME(in *ACME, out *config.ACME, s conversion.Scope) error {
	return autoConvert_v1alpha1_ACME_To_config_ACME(in, out, s)
}

func autoConvert_config_ACME_To_v1alpha1_ACME(in *config.ACME, out *ACME, s conversion.Scope) error {
	out.Email = in.Email
	out.Server = in.Server
	out.PrivateKey = (*string)(unsafe.Pointer(in.PrivateKey))
	return nil
}

// Convert_config_ACME_To_v1alpha1_ACME is an autogenerated conversion function.
func Convert_config_ACME_To_v1alpha1_ACME(in *config.ACME, out *ACME, s conversion.Scope) error {
	return autoConvert_config_ACME_To_v1alpha1_ACME(in, out, s)
}

func autoConvert_v1alpha1_Configuration_To_config_Configuration(in *Configuration, out *config.Configuration, s conversion.Scope) error {
	out.IssuerName = in.IssuerName
	if err := Convert_v1alpha1_ACME_To_config_ACME(&in.ACME, &out.ACME, s); err != nil {
		return err
	}
	out.HealthCheckConfig = (*healthcheckconfig.HealthCheckConfig)(unsafe.Pointer(in.HealthCheckConfig))
	return nil
}

// Convert_v1alpha1_Configuration_To_config_Configuration is an autogenerated conversion function.
func Convert_v1alpha1_Configuration_To_config_Configuration(in *Configuration, out *config.Configuration, s conversion.Scope) error {
	return autoConvert_v1alpha1_Configuration_To_config_Configuration(in, out, s)
}

func autoConvert_config_Configuration_To_v1alpha1_Configuration(in *config.Configuration, out *Configuration, s conversion.Scope) error {
	out.IssuerName = in.IssuerName
	if err := Convert_config_ACME_To_v1alpha1_ACME(&in.ACME, &out.ACME, s); err != nil {
		return err
	}
	out.HealthCheckConfig = (*configv1alpha1.HealthCheckConfig)(unsafe.Pointer(in.HealthCheckConfig))
	return nil
}

// Convert_config_Configuration_To_v1alpha1_Configuration is an autogenerated conversion function.
func Convert_config_Configuration_To_v1alpha1_Configuration(in *config.Configuration, out *Configuration, s conversion.Scope) error {
	return autoConvert_config_Configuration_To_v1alpha1_Configuration(in, out, s)
}
