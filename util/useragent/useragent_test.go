package useragent_test

import (
	"fmt"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/app-autoscaler-cli-plugin/util/useragent"
)

func TestUseragent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Useragent Suite")
}

var _ = Describe("UserAgent", func() {
	It("returns the formatted user agent string", func() {
		ua := useragent.UserAgent("1.2.3", "test-url")

		expected := fmt.Sprintf(
			"app-autoscaler-cli-plugin/1.2.3 (test-url) Go/%s %s/%s",
			runtime.Version(), runtime.GOOS, runtime.GOARCH,
		)

		Expect(ua).To(Equal(expected))
	})

	It("handles empty values", func() {
		ua := useragent.UserAgent("", "")

		expected := fmt.Sprintf(
			"app-autoscaler-cli-plugin/ () Go/%s %s/%s",
			runtime.Version(), runtime.GOOS, runtime.GOARCH,
		)

		Expect(ua).To(Equal(expected))
	})
})
