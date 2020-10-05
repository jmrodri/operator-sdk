package configmap

import (
	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ConfigMap", func() {

	Describe("Package manifest", func() {
		It("Test", func() {
			b := []*apimanifests.Bundle{
				{
					Name: "testbundle",
					Objects: []*unstructured.Unstructured{
						{
							Object: map[string]interface{}{"val1": "val1"},
						},
						{
							Object: map[string]interface{}{"val2": "va2"},
						},
					},
					CSV: &v1alpha1.ClusterServiceVersion{
						Spec: v1alpha1.ClusterServiceVersionSpec{
							Version: version.OperatorVersion{
								Version: semver.SpecVersion,
							},
						},
					},
				},
				{
					Name: "testbundle_2",
					Objects: []*unstructured.Unstructured{
						{
							Object: map[string]interface{}{"val1": "val1"},
						},
						{
							Object: map[string]interface{}{"val2": "va2"},
						},
					},
					CSV: &v1alpha1.ClusterServiceVersion{
						Spec: v1alpha1.ClusterServiceVersionSpec{
							Version: version.OperatorVersion{
								Version: semver.SpecVersion,
							},
						},
					},
				},
			}
			p := apimanifests.PackageManifest{
				PackageName: "test_package_manifest",
				Channels: []apimanifests.PackageChannel{
					{Name: "test_1",
						CurrentCSVName: "test_csv_1",
					},
					{Name: "test_2",
						CurrentCSVName: "test_csv_2",
					},
				},
				DefaultChannelName: "test_channel_name",
			}

			binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(&p, b)
			Expect(err).Should(BeNil())
			val := make(map[string]map[string][]byte)
			cmName := getRegistryConfigMapName(p.PackageName) + "-package"
			val[cmName], err = makeObjectBinaryData(p)
			// Create Bundle ConfigMaps.
			for _, bundle := range b {
				v := bundle.CSV.Spec.Version.String()
				// ConfigMap name containing the bundle's version.
				cmName := getRegistryConfigMapName(p.PackageName) + "-" + k8sutil.FormatOperatorNameDNS1123(v)
				val[cmName], err = makeBundleBinaryData(bundle)
			}
			Expect(err).Should(BeNil())
			Expect(binaryDataByConfigMap).Should(Equal(val))
		})
	})
})
