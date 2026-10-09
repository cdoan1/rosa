// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package idp

import (
	"context"
	"os"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

var hfExitFn = func(code int) { os.Exit(code) }

func runV2() {
	r := rosa.NewRuntime().WithHyperFleet()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()
	cluster, err := hyperfleet.GetCluster(context.Background(), r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		hfExitFn(1)
		return
	}
	if cluster == nil {
		r.Reporter.Errorf("Cluster '%s' not found", clusterKey)
		hfExitFn(1)
		return
	}
	if cluster.Status.Phase != v1alpha1.ClusterPhaseReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		hfExitFn(1)
		return
	}
	r.Reporter.Errorf("Listing identity providers is not supported for Hyperfleet clusters.")
	hfExitFn(1)
}
