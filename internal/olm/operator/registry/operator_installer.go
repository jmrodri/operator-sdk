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
	"context"
	"fmt"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

type OperatorInstaller struct {
	CatalogSourceName     string
	PackageName           string
	StartingCSV           string
	Channel               string
	InstallMode           operator.InstallMode
	CatalogCreator        CatalogCreator
	SupportedInstallModes sets.String

	cfg *operator.Configuration
}

func NewOperatorInstaller(cfg *operator.Configuration) *OperatorInstaller {
	return &OperatorInstaller{cfg: cfg}
}

func (o OperatorInstaller) InstallOperator(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	cs, err := o.CatalogCreator.CreateCatalog(ctx, o.CatalogSourceName)
	if err != nil {
		return nil, fmt.Errorf("create catalog: %v", err)
	}
	log.Infof("Created CatalogSource: %s", cs.GetName())

	// TODO: OLM doesn't appear to propagate the "READY" connection status to the
	// catalogsource in a timely manner even though its catalog-operator reports
	// a connection almost immediately. This condition either needs to be
	// propagated more quickly by OLM or we need to find a different resource to
	// probe for readiness.
	//
	// if err := o.waitForCatalogSource(ctx, cs); err != nil {
	// 	return nil, err
	// }

	// Ensure Operator Group
	fmt.Println("XXX enter ensureOperatorGroup")
	if _, err = o.ensureOperatorGroup(ctx); err != nil {
		fmt.Printf("XXX error from ensureOpreatorGroup: %v\n", err)
		return nil, err
	}
	fmt.Println("XXX returned ensureOperatorGroup")

	var subscription *v1alpha1.Subscription
	// Create Subscription
	if subscription, err = o.createSubscription(ctx, cs); err != nil {
		return nil, err
	}

	// Wait for the Install Plan to be generated
	if err = o.waitForInstallPlan(ctx, subscription); err != nil {
		return nil, err
	}

	// Approve Install Plan for the subscription
	if err = o.approveInstallPlan(ctx, subscription); err != nil {
		return nil, err
	}

	// Wait for successfully installed CSV
	csv, err := o.getInstalledCSV(ctx)
	if err != nil {
		return nil, err
	}

	log.Infof("OLM has successfully installed %q", o.StartingCSV)

	return csv, nil
}

//nolint:unused
func (o OperatorInstaller) waitForCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	catSrcKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("error getting catalog source key: %v", err)
	}

	// verify that catalog source connection status is READY
	catSrcCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := o.cfg.Client.Get(ctx, catSrcKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, catSrcCheck, ctx.Done()); err != nil {
		return fmt.Errorf("catalog source connection is not ready: %v", err)
	}

	return nil
}

func (o OperatorInstaller) ensureOperatorGroup(ctx context.Context) (*v1.OperatorGroup, error) {
	// Check OperatorGroup existence, since we cannot create a second OperatorGroup in namespace.
	og, ogFound, err := o.getOperatorGroup(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("XXX OperatorGroup found? %v\n", ogFound)

	supported := o.SupportedInstallModes

	if supported.Len() == 0 {
		return nil, fmt.Errorf("operator %q is not installable: no supported install modes", o.StartingCSV)
	}

	// --install-mode was given
	if !o.InstallMode.IsEmpty() {
		// TODO: probably remove multinamespace
		if o.InstallMode.InstallModeType == v1alpha1.InstallModeTypeSingleNamespace {
			targetNsSet := sets.NewString(o.InstallMode.TargetNamespaces...)
			if !supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)) && targetNsSet.Has(o.cfg.Namespace) {
				return nil, fmt.Errorf("cannot watch namespace %q: operator %q does not support install mode %q", o.cfg.Namespace, o.StartingCSV, v1alpha1.InstallModeTypeOwnNamespace)
			}
		}
		if o.InstallMode.InstallModeType == v1alpha1.InstallModeTypeSingleNamespace &&
			o.InstallMode.TargetNamespaces[0] == o.cfg.Namespace {
			return nil, fmt.Errorf("use install mode %q to watch operator's namespace %q", v1alpha1.InstallModeTypeOwnNamespace, o.cfg.Namespace)
		}

		supported = supported.Intersection(sets.NewString(string(o.InstallMode.InstallModeType)))
		if supported.Len() == 0 {
			return nil, fmt.Errorf("operator %q does not support install mode %q", o.StartingCSV, o.InstallMode.InstallModeType)
		}

	}
	if !ogFound {
		targetNamespaces, err := o.getTargetNamespaces(supported)
		if err != nil {
			return nil, err
		}
		if og, err = o.createOperatorGroup(ctx, targetNamespaces); err != nil {
			return nil, fmt.Errorf("create operator group: %v", err)
		}
		log.Infof("operatorgroup %q created", og.Name)
	} else if err := o.validateOperatorGroup(*og, supported); err != nil {
		return nil, err
	}

	return og, nil
}

func (o *OperatorInstaller) createOperatorGroup(ctx context.Context, targetNamespaces []string) (*v1.OperatorGroup, error) {
	fmt.Printf("XXX name %v - namespaces %v = namespaces: %v\n", o.cfg.Namespace, o.cfg.Namespace, targetNamespaces)
	og := &v1.OperatorGroup{}
	og.SetName(operator.SDKOperatorGroupName)
	og.SetNamespace(o.cfg.Namespace)
	og.Spec.TargetNamespaces = targetNamespaces

	if err := o.cfg.Client.Create(ctx, og); err != nil {
		fmt.Printf("XXX failed to create og: %v\n", err)
		return nil, err
	}
	return og, nil
}

func (o *OperatorInstaller) validateOperatorGroup(og v1.OperatorGroup, supported sets.String) error {
	ogTargetNs := sets.NewString(og.Spec.TargetNamespaces...)
	imTargetNs := sets.NewString(o.InstallMode.TargetNamespaces...)
	ownNamespaceNs := sets.NewString(o.cfg.Namespace)

	if supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)) && len(og.Spec.TargetNamespaces) == 0 ||
		supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)) && ogTargetNs.Equal(ownNamespaceNs) ||
		supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace)) && ogTargetNs.Equal(imTargetNs) {
		return nil
	}

	switch o.InstallMode.InstallModeType {
	case v1alpha1.InstallModeTypeAllNamespaces, v1alpha1.InstallModeTypeOwnNamespace,
		v1alpha1.InstallModeTypeSingleNamespace:
		return fmt.Errorf("existing operatorgroup %q is not compatible with install mode %q", og.Name, o.InstallMode)
	case "":
		return fmt.Errorf("existing operatorgroup %q is not compatible with any supported package install modes", og.Name)
	}

	return fmt.Errorf("unknown install mode %q", o.InstallMode.InstallModeType)
}

// createOperatorGroup creates an OperatorGroup using package name if an OperatorGroup does not exist.
// If one exists in the desired namespace and it's target namespaces do not match the desired set,
// createOperatorGroup will return an error.
// func (o OperatorInstaller) createOperatorGroup(ctx context.Context) error {
//     fmt.Printf("XXX targetnamespaces: %v\n", o.InstallMode.TargetNamespaces)
//     targetNamespaces := make([]string, len(o.InstallMode.TargetNamespaces), cap(o.InstallMode.TargetNamespaces))
//     copy(targetNamespaces, o.InstallMode.TargetNamespaces)
//     // Check OperatorGroup existence, since we cannot create a second OperatorGroup in namespace.
//     og, ogFound, err := o.getOperatorGroup(ctx)
//     if err != nil {
//         return err
//     }
//     // TODO: we may need to poll for status updates, since status.namespaces may not be updated immediately.
//     if ogFound {
//         // targetNamespaces will always be initialized, but the operator group's namespaces may not be
//         // (required for comparison).
//         if og.Status.Namespaces == nil {
//             og.Status.Namespaces = []string{}
//         }
//         // Simple check for OperatorGroup compatibility: if namespaces are not an exact match,
//         // the user must manage the resource themselves.
//         sort.Strings(og.Status.Namespaces)
//         sort.Strings(targetNamespaces)
//         if !reflect.DeepEqual(og.Status.Namespaces, targetNamespaces) {
//             msg := fmt.Sprintf("namespaces %+q do not match desired namespaces %+q", og.Status.Namespaces, targetNamespaces)
//             if og.GetName() == operator.SDKOperatorGroupName {
//                 return fmt.Errorf("existing SDK-managed operator group's %s, "+
//                     "please clean up existing operators `operator-sdk cleanup` before running package %q", msg, o.PackageName)
//             }
//             return fmt.Errorf("existing operator group %q's %s, "+
//                 "please ensure it has the exact namespace set before running package %q", og.GetName(), msg, o.PackageName)
//         }
//         log.Infof("Using existing operator group %q", og.GetName())
//     } else {
//         // New SDK-managed OperatorGroup.
//         og = newSDKOperatorGroup(o.cfg.Namespace,
//             withTargetNamespaces(targetNamespaces...))
//         log.Info("Creating OperatorGroup")
//         if err = o.cfg.Client.Create(ctx, og); err != nil {
//             return fmt.Errorf("error creating OperatorGroup: %w", err)
//         }
//     }
//     return nil
// }

// getOperatorGroup returns true if an OperatorGroup in the desired namespace was found.
// If more than one operator group exists in namespace, this function will return an error
// since CSVs in namespace will have an error status in that case.
func (o OperatorInstaller) getOperatorGroup(ctx context.Context) (*v1.OperatorGroup, bool, error) {
	ogList := &v1.OperatorGroupList{}
	if err := o.cfg.Client.List(ctx, ogList, client.InNamespace(o.cfg.Namespace)); err != nil {
		return nil, false, err
	}
	if len(ogList.Items) == 0 {
		return nil, false, nil
	}
	if len(ogList.Items) != 1 {
		var names []string
		for _, og := range ogList.Items {
			names = append(names, og.GetName())
		}
		return nil, true, fmt.Errorf("more than one operator group in namespace %s: %+q", o.cfg.Namespace, names)
	}
	return &ogList.Items[0], true, nil
}

func (o OperatorInstaller) createSubscription(ctx context.Context, cs *v1alpha1.CatalogSource) (*v1alpha1.Subscription, error) {
	sub := newSubscription(o.StartingCSV, o.cfg.Namespace,
		withPackageChannel(o.PackageName, o.Channel, o.StartingCSV),
		withCatalogSource(cs.GetName(), o.cfg.Namespace),
		withInstallPlanApproval(v1alpha1.ApprovalManual))

	if err := o.cfg.Client.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("error creating subscription: %w", err)
	}
	log.Infof("Created Subscription: %s", sub.Name)

	return sub, nil
}

func (o OperatorInstaller) getInstalledCSV(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	c, err := olmclient.NewClientForConfig(o.cfg.RESTConfig)
	if err != nil {
		return nil, err
	}

	// BUG(estroz): if namespace is not contained in targetNamespaces,
	// DoCSVWait will fail because the CSV is not deployed in namespace.
	nn := types.NamespacedName{
		Name:      o.StartingCSV,
		Namespace: o.cfg.Namespace,
	}
	log.Infof("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = c.DoCSVWait(ctx, nn); err != nil {
		return nil, fmt.Errorf("error waiting for CSV to install: %w", err)
	}

	// TODO: check status of all resources in the desired bundle/package.
	csv := &v1alpha1.ClusterServiceVersion{}
	if err = o.cfg.Client.Get(ctx, nn, csv); err != nil {
		return nil, fmt.Errorf("error getting installed CSV: %w", err)
	}
	return csv, nil
}

// approveInstallPlan approves the install plan for a subscription, which will
// generate a CSV
func (o OperatorInstaller) approveInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) error {
	ip := v1alpha1.InstallPlan{}

	ipKey := types.NamespacedName{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}

	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := o.cfg.Client.Get(ctx, ipKey, &ip); err != nil {
			return fmt.Errorf("error getting install plan: %v", err)
		}
		// approve the install plan by setting Approved to true
		ip.Spec.Approved = true
		if err := o.cfg.Client.Update(ctx, &ip); err != nil {
			return fmt.Errorf("error approving install plan: %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("Approved InstallPlan %s for the Subscription: %s", ipKey.Name, sub.Name)

	return nil
}

// waitForInstallPlan verifies if an Install Plan exists through subscription status
func (o OperatorInstaller) waitForInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) error {
	subKey := types.NamespacedName{
		Namespace: sub.GetNamespace(),
		Name:      sub.GetName(),
	}

	ipCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := o.cfg.Client.Get(ctx, subKey, sub); err != nil {
			return false, err
		}
		if sub.Status.InstallPlanRef != nil {
			return true, nil
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, ipCheck, ctx.Done()); err != nil {
		return fmt.Errorf("install plan is not available for the subscription %s: %v", sub.Name, err)
	}
	return nil
}

func (o *OperatorInstaller) getTargetNamespaces(supported sets.String) ([]string, error) {
	switch {
	case supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)):
		return nil, nil
	case supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)):
		return []string{o.cfg.Namespace}, nil
	case supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace)):
		if len(o.InstallMode.TargetNamespaces) != 1 {
			return nil, fmt.Errorf("install mode %q requires explicit target namespace", v1alpha1.InstallModeTypeSingleNamespace)
		}
		return o.InstallMode.TargetNamespaces, nil
	default:
		return nil, fmt.Errorf("no supported install modes")
	}
}

func getSupportedInstallModes(csvInstallModes []v1alpha1.InstallMode) sets.String {
	supported := sets.NewString()
	for _, im := range csvInstallModes {
		if im.Supported {
			supported.Insert(string(im.Type))
		}
	}
	return supported
}
