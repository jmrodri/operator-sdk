package olm

import (
	"fmt"
	"testing"
)

func TestNewManager(t *testing.T) {
	m, err := NewBundleManager("1.3")
	if err != nil {
		t.Fatal()
	}
	fmt.Println(m.version)
}
