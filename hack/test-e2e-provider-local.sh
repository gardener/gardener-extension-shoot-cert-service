#!/bin/bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit

repo_root="$(readlink -f $(dirname ${0})/..)"

if [[ ! -d "$repo_root/gardener" ]]; then
  git clone https://github.com/gardener/gardener.git
fi

gardener_version=$(go list -m -f '{{.Version}}' github.com/gardener/gardener)
gardener_version='50828b680ca0' # TODO(martinweindel): remove this line as soon as the gardener version is updated
cd "$repo_root/gardener"
git checkout "$gardener_version"
source "$repo_root/gardener/hack/ci-common.sh"

echo ">>>>>>>>>>>>>>>>>>>> kind-operator-up"
make kind-operator-up
trap '{
  cd "$repo_root/gardener"
  export_artifacts "gardener-local"
  make kind-operator-down
}' EXIT
export KUBECONFIG=$repo_root/gardener/example/provider-local/seed-operator/base/kubeconfig
echo "<<<<<<<<<<<<<<<<<<<< kind-operator-up done"

echo ">>>>>>>>>>>>>>>>>>>> operator-up"
make operator-up
echo "<<<<<<<<<<<<<<<<<<<< operator-up done"

echo ">>>>>>>>>>>>>>>>>>>> operator-seed-up"
make operator-seed-up
echo "<<<<<<<<<<<<<<<<<<<< operator-seed-up done"

cd $repo_root

echo ">>>>>>>>>>>>>>>>>>>> extension-up"
make extension-up
echo "<<<<<<<<<<<<<<<<<<<< extension-up done"

export REPO_ROOT=$repo_root

# reduce flakiness in contended pipelines
export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=5s
export GOMEGA_DEFAULT_EVENTUALLY_POLLING_INTERVAL=200ms
# if we're running low on resources, it might take longer for tested code to do something "wrong"
# poll for 5s to make sure, we're not missing any wrong action
export GOMEGA_DEFAULT_CONSISTENTLY_DURATION=5s
export GOMEGA_DEFAULT_CONSISTENTLY_POLLING_INTERVAL=200ms

ginkgo --timeout=1h --v --progress "$@" $repo_root/test/e2e/...

cd "$repo_root/gardener"
make kind-operator-down