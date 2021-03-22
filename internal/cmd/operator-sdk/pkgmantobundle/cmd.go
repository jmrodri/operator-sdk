// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pkgmantobundle

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

// operator-sdk pkgman-to-bundle <packagemanifestdir> [--build-image=]  [--output-dir=] [--image-base=] [--build-cmd=]
type ptbCmd struct {
	buildImage         bool
	packagemanifestdir string
	outputDir          string
	imageBase          string
	buildCmd           string
}

func NewCmd() *cobra.Command {
	var timeout time.Duration

	c := &ptbCmd{}

	// i := bundle.NewInstall(cfg)
	cmd := &cobra.Command{
		Use:   "pkgman-to-bundle <packagemanifestdir>",
		Short: "Migrate the given packagemanifest to one or more bundles",
		Args:  cobra.ExactArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// return cfg.Load()
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			_, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			c.packagemanifestdir = args[0]

			// i.BundleImage = args[0]
			//
			// _, err := i.Run(ctx)
			// if err != nil {
			//     logrus.Fatalf("Failed to run bundle: %v\n", err)
			// }

			// so we want to take the packagemanifestdir, verify it is a directory. Verify it has manifests in it.g
			// for each directory we want to create a bundle from it.
			//
			// type WalkFunc func(path string, info fs.FileInfo, err error) error
			pkg, bundles, err := apimanifests.GetManifestsDir(c.packagemanifestdir)
			if err != nil {
				logrus.Errorf("Error getting packagemanifest: %v\n", err)
			}
			if len(bundles) == 0 {
				logrus.Error("no packages found")
			}
			if pkg == nil || pkg.PackageName == "" {
				logrus.Error("no package manifest found")
			}

			logrus.Info("looping through bundles")
			for _, b := range bundles {
				logrus.Info(b.Name)
				logrus.Info(b.Package)
				logrus.Infof("Default Channel: %v\n", b.DefaultChannel)
				for _, c := range b.Channels {
					logrus.Infof("Channel: %v\n", c)
				}
			}

			if err := os.MkdirAll(c.outputDir, 0755); err != nil {
				logrus.Errorf("Error making outputdir: %v\n", err)
			}

			if err := runManifests(c.packagemanifestdir, c.outputDir); err != nil {
				logrus.Errorf("Error generating manifests: %v\n", err)
			}

			logrus.Info("Running pkgman-to-bundle")
		},
	}
	cmd.Flags().SortFlags = false
	// cfg.BindFlags(cmd.PersistentFlags())
	// i.BindFlags(cmd.Flags())

	// cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "install timeout")
	c.addFlagsTo(cmd.Flags())

	return cmd
}

func runManifests(inputDir, outputDir string) error {
	col := &collector.Manifests{}
	col.UpdateFromDir(inputDir)

	objs := genutil.GetManifestObjects(col)
	dir := filepath.Join(outputDir, "manifests")
	if err := genutil.WriteObjectsToFiles(dir, objs...); err != nil {
		return err
	}
	return nil
}

func runMetadata() error {
	return nil
}

func (c *ptbCmd) addFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVar(&c.buildImage, "build-image", false, "if true, we will build images; if false, output to bundle directory or value specified by output-dir; defaults to false")
	fs.StringVar(&c.outputDir, "output-dir", "bundle", "the directory to write the bundle to, if not present defaults to bundle directory")
	fs.StringVar(&c.imageBase, "image-base", "", "the base container name for the bundle images; e.g. quay.io/example/memcached-operator-bundle")
	fs.StringVar(&c.buildCmd, "build-cmd", "", "build command override e.g. podman build -t quay.io/example/bundle ...")
}
