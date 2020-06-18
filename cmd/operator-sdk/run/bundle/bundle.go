package bundle

import (
	"fmt"
	"log"

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
	fmt.Println(c.BundleImage)
	fmt.Println(c.IndexImage)
	fmt.Println(c.Namespace)
	fmt.Println(c.InstallMode)
	return nil
}

func (c *BundleCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a bundle image is a required argument")
	}
	return nil
}

func NewCmd() *cobra.Command {
	c := &BundleCmd{}

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Run an Operator organized in bundle format with OLM",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validate(args); err != nil {
				return fmt.Errorf("invalid command args: %v", err)
			}

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
