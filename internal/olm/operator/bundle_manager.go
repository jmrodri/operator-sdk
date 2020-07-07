package olm

import apimanifests "github.com/operator-framework/api/pkg/manifests"

type bundleManager struct {
	*operatorManager

	version string
	bundles []*apimanifests.Bundle
}

func NewManager(version string) (*bundleManager, error) {
	m := &bundleManager{
		version: version,
	}

	return m, nil
}
