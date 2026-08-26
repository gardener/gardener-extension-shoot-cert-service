#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

go get github.com/gardener/cert-management/pkg/apis@$(go list -m -f "{{.Version}}" github.com/gardener/cert-management/pkg/apis)

src_dir=$(go list -m -f '{{.Dir}}' github.com/gardener/cert-management/pkg/apis)

cp "${src_dir}/cert/crds/"* ./pkg/controller/extension/shared/assets
