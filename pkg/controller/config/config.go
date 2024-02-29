// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/gardener/gardener-extension-shoot-cert-service/pkg/apis/config"
)

// Config contains configuration for the shoot cert service.
type Config struct {
	config.Configuration
}
