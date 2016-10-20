package command_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBuildpackUsage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buildpack-Usage Suite")
}
