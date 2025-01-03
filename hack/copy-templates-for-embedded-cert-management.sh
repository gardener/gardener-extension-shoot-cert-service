#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT="$(dirname $0)"/..

SEED_TEMPLATES=$PROJECT_ROOT/charts/internal/shoot-cert-management-seed/templates
SHOOT_TEMPLATES=$PROJECT_ROOT/charts/internal/shoot-cert-management-shoot/templates
DEST_DIR=$PROJECT_ROOT/charts/internal/embedded-cert-management/templates

mkdir -p $DEST_DIR

echo "Copying templates for embedded-cert-managemen charts"

cp $SEED_TEMPLATES/0helpers.tpl $DEST_DIR
cp $SEED_TEMPLATES/ca-certificats-configmap.yaml $DEST_DIR
cp $SEED_TEMPLATES/deployment.yaml $DEST_DIR
cp $SEED_TEMPLATES/issuer.yaml $DEST_DIR
cp $SEED_TEMPLATES/rbac.yaml $DEST_DIR/rbac-role.yaml
cp $SEED_TEMPLATES/service.yaml $DEST_DIR
cp $SEED_TEMPLATES/serviceaccount.yaml $DEST_DIR
cp $SEED_TEMPLATES/vpa.yaml $DEST_DIR

cp $SHOOT_TEMPLATES/1helpers.tpl $DEST_DIR
cp $SHOOT_TEMPLATES/crds-v1.yaml $DEST_DIR
cp $SHOOT_TEMPLATES/rbac.yaml $DEST_DIR/rbac-clusterrole.yaml
cp $SHOOT_TEMPLATES/cert-management-role.yaml $DEST_DIR
cp $SHOOT_TEMPLATES/cert-management-rolebinding.yaml $DEST_DIR
