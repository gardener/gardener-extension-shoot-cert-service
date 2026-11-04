// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package install_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstall(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service API Install Suite")
}
