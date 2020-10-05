package configmap

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfigMap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigMap")
}
