package olm

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {

	m, err := NewBundleManager("1.3", "foobarns", "docker.io/jmrodri/image:latest", "", "OwnNamespace")
	if err != nil {
		t.Fail()
	}
	fmt.Println(m.version)

}
