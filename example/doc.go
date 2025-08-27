// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "extension-generator --name=extension-shoot-cert-service --provider-type=shoot-cert-service --component-category=extension --extension-oci-repository=europe-docker.pkg.dev/gardener-project/public/charts/gardener/extensions/shoot-cert-service:$(cat ../VERSION) --destination=./extension/base/extension.yaml"
//go:generate sh -c "$TOOLS_BIN_DIR/kustomize build ./extension -o ./extension.yaml"

package example
