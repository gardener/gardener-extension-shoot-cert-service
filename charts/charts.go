// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package charts

import (
	"embed"
)

// Internal contains the internal charts
//
//go:embed internal
var Internal embed.FS

// ChartsPath is the path to the charts
const ChartsPath = "internal"
