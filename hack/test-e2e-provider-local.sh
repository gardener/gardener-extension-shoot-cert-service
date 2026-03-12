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
cd "$repo_root/gardener"
git checkout "$gardener_version"
source "$repo_root/gardener/hack/ci-common.sh"

echo ">>>>>>>>>>>>>>>>>>>> kind-single-node-up"
make kind-single-node-up
trap '{
  cd "$repo_root/gardener"
  export_artifacts "gardener-local"
  make kind-single-node-down
}' EXIT
export KUBECONFIG=$repo_root/gardener/dev-setup/gardenlet/components/kubeconfigs/seed-local/kubeconfig
echo "<<<<<<<<<<<<<<<<<<<< kind-single-node-up done"

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

ginkgo --timeout=30m --v --show-node-events "$@" $repo_root/test/e2e/...

echo ">>>>>>>>>>>>>>>>>>>> kind-single-node-down"
cd "$repo_root/gardener"
make kind-single-node-down
echo "<<<<<<<<<<<<<<<<<<<< kind-single-node-down done"
