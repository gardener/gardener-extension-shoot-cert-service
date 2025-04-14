// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate extension-generator --name=extension-shoot-cert-service --provider-type=shoot-cert-service --component-category=extension --extension-oci-repository=local-skaffold/gardener-extension-shoot-cert-service/charts/extension:v0.0.0 --destination=$PWD/extension.yaml

package operator
