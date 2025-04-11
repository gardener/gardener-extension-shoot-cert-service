#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o pipefail

repo_root="$(readlink -f $(dirname ${0})/..)"

export KUBECONFIG=$repo_root/gardener/example/provider-local/seed-operator/base/kubeconfig

kubectl delete ns pebble --ignore-not-found
