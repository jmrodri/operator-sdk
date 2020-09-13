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

package registry

import (
	// "context"

	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("OperatorInstaller", func() {
	Describe("InstallOperator", func() {
		// TODO: fill this in once run bundle is done
	})

	Describe("ensureOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "testns",
				},
			}

			// setup supported install modes
			modes := []v1alpha1.InstallMode{
				{
					Type:      v1alpha1.InstallModeTypeSingleNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeOwnNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeAllNamespaces,
					Supported: true,
				},
			}
			oi.SupportedInstallModes = operator.GetSupportedInstallModes(modes)
		})
		It("should return an error when problems finding OperatorGroup", func() {
			oi.cfg.Client = fake.NewFakeClient()
			grp, err := oi.ensureOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("should return an error if there are no supported modes", func() {
			oi.SupportedInstallModes = operator.GetSupportedInstallModes([]v1alpha1.InstallMode{})
			grp, err := oi.ensureOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("no supported install modes"))
		})
		// It("should return an error if OwnNamespace is used and target does not match", func() {
		//     oi.InstallMode.Set(string(v1alpha1.InstallModeTypeOwnNamespace))
		//     oi.InstallMode.TargetNamespaces = []string{"notownns"}
		//     og, err := oi.ensureOperatorGroup(context.TODO())
		//     Expect(og).To(BeNil())
		//     Expect(err).ToNot(BeNil())
		//     Expect(err.Error()).Should(ContainSubstring("use install mode \"OwnNamespace\""))
		// })
		Context("with no existing OperatorGroup", func() {
			Context("given SingleNamespace", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
					oi.InstallMode.TargetNamespaces = []string{"anotherns"}
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(og).ToNot(BeNil())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(og.Spec.TargetNamespaces).To(Equal([]string{"anotherns"}))
				})
				It("should return an error if target matches operator ns", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
					oi.InstallMode.TargetNamespaces = []string{"testns"}
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ToNot(BeNil())
					Expect(og).To(BeNil())
					Expect(err.Error()).Should(ContainSubstring("use install mode \"OwnNamespace\""))
				})
			})
			Context("given OwnNamespace", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeOwnNamespace))
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(og).ToNot(BeNil())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(len(og.Spec.TargetNamespaces)).To(Equal(1))
				})
			})
			Context("given AllNamespaces", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeAllNamespaces))
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(og).ToNot(BeNil())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(len(og.Spec.TargetNamespaces)).To(Equal(0))
				})
			})
		})
		Context("with an existing OperatorGroup", func() {
			Context("given AllNamespaces", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeAllNamespaces))
				})
				It("should return nil for AllNamespaces with empty targets", func() {
					// context, client, name, ns, targets
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og", "testns")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for AllNamespaces when target is not empty", func() {
					// context, client, name, ns, targets
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "incompatiblens")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("is not compatible"))
					Expect(og).To(BeNil())
				})
			})
			Context("given OwnNamespace", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeOwnNamespace))
				})
				It("should return nil for OwnNamespace when target matches operator", func() {
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "testns")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for OwnNamespace when target does not match operator", func() {
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "incompatiblens")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("is not compatible"))
					Expect(og).To(BeNil())
				})
			})
			Context("given SingleNamespace", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
				})
				It("should return nil for SingleNamespace when target differs from operator", func() {
					oi.InstallMode.TargetNamespaces = []string{"anotherns"}
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "anotherns")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for SingleNamespace when target matches operator", func() {
					oi.InstallMode.TargetNamespaces = []string{"testns"}
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "testns")
					og, err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("use install mode \"OwnNamespace\""))
					Expect(og).To(BeNil())
				})
			})
		})
	})

	Describe("createOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "testnamespace",
				},
			}
		})
		It("should return an error if OperatorGroup already exists", func() {
			_ = createOperatorGroupHelper(context.TODO(), client,
				operator.SDKOperatorGroupName, "testnamespace")

			og, err := oi.createOperatorGroup(context.TODO(), nil)
			Expect(og).Should(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("should create the OperatorGroup", func() {
			og, err := oi.createOperatorGroup(context.TODO(), nil)
			Expect(og).ShouldNot(BeNil())
			Expect(og.Name).To(Equal(operator.SDKOperatorGroupName))
			Expect(og.Namespace).To(Equal("testnamespace"))
			Expect(err).To(BeNil())
		})
	})

	Describe("getOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "atestns",
				},
			}
		})
		It("should return an error if no OperatorGroups exist", func() {
			oi.cfg.Client = fake.NewFakeClient()
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).To(HaveOccurred())
		})
		It("should return nothing if namespace does not match", func() {
			oi.cfg.Namespace = "fakens"
			_ = createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).Should(BeNil())
		})
		It("should return an error when more than OperatorGroup found", func() {
			_ = createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			_ = createOperatorGroupHelper(context.TODO(), client, "og2", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeTrue())
			Expect(err).Should(HaveOccurred())
		})
		It("should return list of OperatorGroups", func() {
			og := createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).ShouldNot(BeNil())
			Expect(grp.Name).To(Equal(og.Name))
			Expect(grp.Namespace).To(Equal(og.Namespace))
			Expect(found).To(BeTrue())
			Expect(err).Should(BeNil())
		})
	})

	Describe("createSubscription", func() {
	})

	Describe("getTargetNamespaces", func() {
		var (
			oi        OperatorInstaller
			supported sets.String
		)
		BeforeEach(func() {
			oi = OperatorInstaller{
				cfg: &operator.Configuration{},
			}
			supported = sets.NewString()
		})
		It("should return an error when nothing is supported", func() {
			target, err := oi.getTargetNamespaces(supported)
			Expect(target).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no supported install modes"))
		})
		It("should return nothing when AllNamespaces is supported", func() {
			supported.Insert(string(v1alpha1.InstallModeTypeAllNamespaces))
			target, err := oi.getTargetNamespaces(supported)
			Expect(target).To(BeNil())
			Expect(err).To(BeNil())
		})
		It("should return operator's namespace when OwnNamespace is supported", func() {
			oi.cfg.Namespace = "test-ns"
			supported.Insert(string(v1alpha1.InstallModeTypeOwnNamespace))
			target, err := oi.getTargetNamespaces(supported)
			Expect(len(target)).To(Equal(1))
			Expect(target[0]).To(Equal("test-ns"))
			Expect(err).To(BeNil())
		})
		It("should return configured namespace when SingleNamespace is passed in", func() {

			oi.InstallMode = operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeSingleNamespace,
				TargetNamespaces: []string{"test-ns"},
			}

			supported.Insert(string(v1alpha1.InstallModeTypeSingleNamespace))
			target, err := oi.getTargetNamespaces(supported)
			Expect(len(target)).To(Equal(1))
			Expect(target[0]).To(Equal("test-ns"))
			Expect(err).To(BeNil())
		})
	})
})

func createOperatorGroupHelper(ctx context.Context, c crclient.Client, name, namespace string, targetNamespaces ...string) v1.OperatorGroup {
	og := v1.OperatorGroup{}
	og.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("OperatorGroup"))
	og.SetName(name)
	og.SetNamespace(namespace)
	og.Spec.TargetNamespaces = targetNamespaces
	og.Status.Namespaces = targetNamespaces
	if c != nil {
		ExpectWithOffset(1, c.Create(ctx, &og)).Should(Succeed())
	}
	return og
}
