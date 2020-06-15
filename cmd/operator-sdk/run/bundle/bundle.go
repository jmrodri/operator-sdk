package bundle

import (
	"fmt"
	"log"

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
	fs.StringVar(&c.IndexImage, "index-image", "",
		"index image")
	fs.StringVar(&c.InstallMode, "install-mode", "",
		"the install mode")
}

// fs.StringVar(&c.Namespace, "namespace", "",
//     "Specifies the namespace to install the operator. It will default to the KUBECONFIG context")

func (c *BundleCmd) Run() error {

	fmt.Println("Hello from bundle cmd")
	fmt.Println(c.IndexImage)
	fmt.Println(c.Namespace)
	fmt.Println(c.InstallMode)
	return nil
}

func NewCmd() *cobra.Command {
	c := &BundleCmd{}
	c.BundleImage = "bundle:foo"

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Run an Operator organized in bundle format with OLM",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.Run(); err != nil {
				log.Fatalf("Failed to run operator: %v", err)
			}
			return nil
		},
	}

	c.AddToFlagSet(cmd.Flags())

	return cmd
}
