// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestShoot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extension Shoot Controller Suite")
}
