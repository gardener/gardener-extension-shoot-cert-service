package extension

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDNSManagement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extension Controller Suite")
}
