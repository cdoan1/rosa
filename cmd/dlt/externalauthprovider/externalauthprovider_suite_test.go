package externalauthprovider

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteExternalAuthProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete external authentication provider suite")
}
