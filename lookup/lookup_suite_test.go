package lookup_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLookup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lookup Suite")
}
