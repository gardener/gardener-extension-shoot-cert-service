// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"crypto/sha256"
	"fmt"

	certv1alpha1 "github.com/gardener/cert-management/pkg/apis/cert/v1alpha1"
	"github.com/gardener/cert-management/pkg/cert/source"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func (d *Deployer) collectIssuers() ([]Issuer, error) {
	gardenIssuer := Issuer{Name: d.values.ExtensionConfig.IssuerName}
	if acme := d.values.ExtensionConfig.ACME; acme != nil {
		gardenIssuer.ACME = &ACME{
			Email:                      acme.Email,
			Server:                     acme.Server,
			PrivateKey:                 acme.PrivateKey,
			SkipDNSChallengeValidation: ptr.Deref(acme.SkipDNSChallengeValidation, false),
		}
	}
	if ca := d.values.ExtensionConfig.CA; ca != nil {
		gardenIssuer.CA = &CA{
			Certificate:    ca.Certificate,
			CertificateKey: ca.CertificateKey,
		}
	}

	issuerList := []Issuer{gardenIssuer}

	if !d.values.ShootDeployment {
		return issuerList, nil
	}

	for _, issuer := range d.values.CertConfig.Issuers {
		if issuer.Name == d.values.ExtensionConfig.IssuerName {
			continue
		}

		acme := &ACME{
			Email:  issuer.Email,
			Server: issuer.Server,
		}
		if issuer.PrivateKeySecretName != nil {
			var err error
			acme.PrivateKeySecretName, err = d.lookupReferencedSecret(*issuer.PrivateKeySecretName)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup referenced private key secret for issuer %s: %w", issuer.Name, err)
			}
		}
		if issuer.ExternalAccountBinding != nil {
			secretName, err := d.lookupReferencedSecret(issuer.ExternalAccountBinding.KeySecretName)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup referenced private key secret for issuer %s: %w", issuer.Name, err)
			}
			acme.ExternalAccountBinding = &ExternalAccountBinding{
				KeyID:         issuer.ExternalAccountBinding.KeyID,
				KeySecretName: secretName,
			}
		}
		if issuer.SkipDNSChallengeValidation != nil && *issuer.SkipDNSChallengeValidation {
			acme.SkipDNSChallengeValidation = true
		}
		if issuer.Domains != nil && len(issuer.Domains.Include)+len(issuer.Domains.Exclude) > 0 {
			acme.Domains = &Domains{}
			if issuer.Domains.Include != nil {
				acme.Domains.Include = issuer.Domains.Include
			}
			if issuer.Domains.Exclude != nil {
				acme.Domains.Exclude = issuer.Domains.Exclude
			}
		}

		modelIssuer := Issuer{
			Name: issuer.Name,
			ACME: acme,
		}
		if issuer.RequestsPerDayQuota != nil {
			modelIssuer.RequestsPerDayQuota = *issuer.RequestsPerDayQuota
		}
		if len(issuer.PrecheckNameservers) > 0 {
			modelIssuer.PrecheckNameservers = issuer.PrecheckNameservers
		}
		issuerList = append(issuerList, modelIssuer)
	}

	return issuerList, nil
}

func (d *Deployer) issuersChecksum() (string, error) {
	issuers, err := d.collectIssuers()
	if err != nil {
		return "", err
	}
	issuersData, err := yaml.Marshal(issuers)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(issuersData)), nil
}

func (d *Deployer) createIssuers() ([]client.Object, error) {
	var objects []client.Object

	issuers, err := d.collectIssuers()
	if err != nil {
		return nil, err
	}

	for _, issuer := range issuers {
		if issuer.ACME != nil && issuer.ACME.PrivateKey != nil {
			objects = append(objects, d.secretACME(issuer))
		}
		if issuer.CA != nil {
			objects = append(objects, d.secretCA(issuer))
		}
		objects = append(objects, d.createIssuer(issuer))
	}
	return objects, nil
}

func (d *Deployer) secretACME(issuer Issuer) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("extension-shoot-cert-service-issuer-%s", issuer.Name),
			Namespace: d.values.Namespace,
		},
		Data: map[string][]byte{
			"email":      []byte(issuer.ACME.Email),
			"privateKey": []byte(*issuer.ACME.PrivateKey),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func (d *Deployer) secretCA(issuer Issuer) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("extension-shoot-cert-service-issuer-%s-ca", issuer.Name),
			Namespace: d.values.Namespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte(issuer.CA.Certificate),
			"tls.key": []byte(issuer.CA.CertificateKey),
		},
		Type: corev1.SecretTypeTLS,
	}
}

func (d *Deployer) createIssuer(input Issuer) *certv1alpha1.Issuer {
	issuer := &certv1alpha1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: d.values.Namespace,
		},
		Spec: certv1alpha1.IssuerSpec{
			ACME: d.createACMESpec(input),
			CA:   d.createCASpec(input),
		},
	}
	if d.values.CertClass != "" {
		issuer.Annotations = map[string]string{source.AnnotClass: d.values.CertClass}
	}
	if input.RequestsPerDayQuota > 0 {
		issuer.Spec.RequestsPerDayQuota = &input.RequestsPerDayQuota
	}
	return issuer
}

func (d *Deployer) createACMESpec(issuer Issuer) *certv1alpha1.ACMESpec {
	input := issuer.ACME
	if input == nil {
		return nil
	}
	secretName := input.PrivateKeySecretName
	if secretName == "" {
		secretName = fmt.Sprintf("extension-shoot-cert-service-issuer-%s", issuer.Name)
	}
	acme := &certv1alpha1.ACMESpec{
		Email:  input.Email,
		Server: input.Server,
		PrivateKeySecretRef: &corev1.SecretReference{
			Name:      secretName,
			Namespace: d.values.Namespace,
		},
		AutoRegistration:    input.PrivateKeySecretName == "" && input.PrivateKey == nil,
		PrecheckNameservers: issuer.PrecheckNameservers,
	}
	if input.ExternalAccountBinding != nil {
		acme.ExternalAccountBinding = &certv1alpha1.ACMEExternalAccountBinding{
			KeyID: input.ExternalAccountBinding.KeyID,
			KeySecretRef: &corev1.SecretReference{
				Name:      input.ExternalAccountBinding.KeySecretName,
				Namespace: d.values.Namespace,
			},
		}
	}
	if input.SkipDNSChallengeValidation {
		acme.SkipDNSChallengeValidation = ptr.To(input.SkipDNSChallengeValidation)
	}
	if input.Domains != nil {
		acme.Domains = &certv1alpha1.DNSSelection{
			Include: input.Domains.Include,
			Exclude: input.Domains.Exclude,
		}
	}

	return acme
}

func (d *Deployer) createCASpec(issuer Issuer) *certv1alpha1.CASpec {
	if issuer.CA == nil {
		return nil
	}
	return &certv1alpha1.CASpec{
		PrivateKeySecretRef: &corev1.SecretReference{
			Name:      fmt.Sprintf("extension-shoot-cert-service-issuer-%s-ca", issuer.Name),
			Namespace: d.values.Namespace,
		},
	}
}

func (d *Deployer) lookupReferencedSecret(refname string) (string, error) {
	if !d.values.ShootDeployment {
		return "invalid", fmt.Errorf("only shoot deployment supports additional issuers")
	}
	for _, ref := range d.values.Resources {
		if ref.Name == refname {
			if ref.ResourceRef.Kind != "Secret" {
				return "invalid-kind", fmt.Errorf("invalid referenced resource, expected kind Secret, not %s: %s", ref.ResourceRef.Kind, refname)
			}
			return v1beta1constants.ReferencedResourcesPrefix + ref.ResourceRef.Name, nil
		}
	}

	return "invalid", fmt.Errorf("invalid referenced resource: %s", refname)
}
