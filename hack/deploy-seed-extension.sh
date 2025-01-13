#!/bin/bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit

function apply() {
  cd "$repo_root/test/e2e/garden/assets/charts"

  cat <<EOF | kubectl --kubeconfig $virtual_kubeconfig apply -f -
---
apiVersion: core.gardener.cloud/v1
helm:
  rawChart: $(tar cfvz - shoot-cert-service-seed |base64 -w 0)
kind: ControllerDeployment
metadata:
  name: shoot-cert-service-seed
---
apiVersion: core.gardener.cloud/v1beta1
kind: ControllerRegistration
metadata:
  name: shoot-cert-service-seed
spec:
  deployment:
    deploymentRefs:
    - name: shoot-cert-service-seed
    policy: Always
EOF
}

function delete() {
  kubectl --kubeconfig $virtual_kubeconfig delete controllerregistration shoot-cert-service-seed --ignore-not-found
  kubectl --kubeconfig $virtual_kubeconfig delete controllerdeployment shoot-cert-service-seed --ignore-not-found
}

if [ $# -eq 0 ]; then
  echo "Usage: $0 {apply|delete}"
  exit 1
fi

repo_root="$(readlink -f $(dirname ${0})/..)"

runtime_kubeconfig=$repo_root/gardener/example/provider-local/seed-operator/base/kubeconfig
virtual_kubeconfig=$(mktemp)
kubectl --kubeconfig $runtime_kubeconfig -n garden  get secret gardener -ojsonpath={.data.kubeconfig} |base64 -d > $virtual_kubeconfig

if [ "$1" == "delete" ]; then
  delete
elif [ "$1" == "apply" ]; then
  apply
else
  echo "Invalid argument: $1"
  echo "Usage: $0 {apply|delete}"
  exit 1
fi

rm $virtual_kubeconfig