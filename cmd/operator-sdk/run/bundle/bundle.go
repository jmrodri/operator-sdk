package bundle

import (
	"context"
	"fmt"
	"log"
	"time"

	olm "github.com/operator-framework/operator-sdk/internal/olm/operator"
	k8sinternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type BundleCmd struct {
	BundleImage string
	IndexImage  string
	Namespace   string
	InstallMode string

	/*
		bundle <bundle-image>
			[--index-image=] [--namespace=]
			[--install-mode=(AllNamespace|OwnNamespace|SingleNamespace=)]
	*/
}

func (c *BundleCmd) AddToFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&c.Namespace, "namespace", "",
		"Specifies the namespace to install the operator. It will default to the KUBECONFIG context")
	fs.StringVar(&c.IndexImage, "index-image", "quay.io/operator-framework/upstream-opm-builder:latest",
		"index image")
	fs.StringVar(&c.InstallMode, "install-mode", "",
		"the install mode")
}

// fs.StringVar(&c.Namespace, "namespace", "",
//     "Specifies the namespace to install the operator. It will default to the KUBECONFIG context")

func (c *BundleCmd) Run() error {

	fmt.Println("Hello from bundle cmd")
	fmt.Printf("Bundle image is [%v]\n", c.BundleImage)
	fmt.Printf("Index image is [%v]\n", c.IndexImage)
	fmt.Printf("Namespace is [%v]\n", c.Namespace)
	fmt.Printf("Install Mode is [%v]\n", c.InstallMode)

	m, err := olm.NewBundleManager("1.3", c.Namespace, c.BundleImage, c.IndexImage, c.InstallMode)
	if err != nil {
		fmt.Printf("error %v", err)
	}

	// TODO: consider reusing OperatorCmd
	defaultTimeout := time.Minute * 2
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	err = m.Run(ctx)
	if err != nil {
		fmt.Printf("error %v", err)
	}

	return nil
}

func (c *BundleCmd) validate() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("a BUNDLE_IMAGE is a required argument")
		}
		return nil
	}
}

func NewCmd() *cobra.Command {
	c := &BundleCmd{}

	cmd := &cobra.Command{
		Use:   "bundle BUNDLE_IMAGE",
		Short: "Run an Operator organized in bundle format with OLM",
		Args:  c.validate(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// assign the bundle image
			c.BundleImage = args[0]

			if !cmd.Flags().Changed("namespace") {
				fmt.Println("namespace changed")
				// we should probably allow folks to pass in their kubeconfig
				// path
				_, defaultNamespace, err := k8sinternal.GetKubeconfigAndNamespace("")
				if err != nil {
					return fmt.Errorf("error getting kubeconfig and default namespace: %v", err)
				}
				c.Namespace = defaultNamespace
			}
			if err := c.Run(); err != nil {
				log.Fatalf("Failed to run operator: %v", err)
			}
			return nil
		},
	}

	c.AddToFlagSet(cmd.Flags())

	return cmd
}
