package olm

import (
	"context"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
)

type BundleManager struct {
	*operatorManager

	version string
	bundles []*apimanifests.Bundle
}

func (b *BundleManager) Run(ctx context.Context) error {
	/*
	 * create Pod spec for registryImage
	 * image: = registryImage value
	 * set entrypoint to be:
	 * /bin/bash -c ‘ \
	 *		  /bin/mkdir -p /database && \
	 *		  /bin/opm registry add   -d /database/index.db -b {.BundleImage} && \
	 *		  /bin/opm registry serve -d /database/index.db’
	 * IF pod fails, clean up and error out. Capture pod logs
	 * Create GRPC CatalogSource, point to :50051
	 * Create OperatorGroup (see note about installmode)
	 */

	/*
	 * Create Subscription
	 * Verify operator is installed
	 */
	return nil
}
