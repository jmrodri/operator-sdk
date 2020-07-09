package olm

import apimanifests "github.com/operator-framework/api/pkg/manifests"

type BundleManager struct {
	*operatorManager

	version string
	bundles []*apimanifests.Bundle
}

func NewBundleManager(version string) (*BundleManager, error) {
	m := &BundleManager{
		version: version,
	}

	return m, nil
}
