package tests

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("VM delete protection", func() {
	BeforeEach(func() {
		waitUntilDeployed()
	})
	It("should not allow to delete a VM if the protection is enabled", func() {

	})

})
