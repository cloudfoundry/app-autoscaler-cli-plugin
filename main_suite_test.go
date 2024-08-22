package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func TestAppAutoScaler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App-AutoScaler Suite")
}

var validPluginPath string

var _ = BeforeSuite(func() {
	var err error
	validPluginPath, err = Build("code.cloudfoundry.org/app-autoscaler-cli-plugin")
	Expect(err).NotTo(HaveOccurred())
})

// gexec.Build leaves a compiled binary behind in /tmp.
var _ = AfterSuite(func() {
	CleanupBuildArtifacts()
})
