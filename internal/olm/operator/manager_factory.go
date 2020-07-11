package olm

func NewBundleManager(version, ns, bundleImage, indexImage, installMode string) (*BundleManager, error) {
	m := &BundleManager{
		version: version,
	}

	return m, nil
}

/*
func NewPackageManifestManager() error {
	return nil
}
*/
