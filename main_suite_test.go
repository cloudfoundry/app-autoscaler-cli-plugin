package main_test

import (
  . "github.com/onsi/ginkgo/v2"
  . "github.com/onsi/gomega"
  "testing"
)

func TestMain(t *testing.T) {
  RegisterFailHandler(Fail)
  RunSpecs(t, "Main cli Suite")
}
