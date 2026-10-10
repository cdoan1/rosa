// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfExitFn                      = func(code int) { os.Exit(code) }
	hfConfirmExternalAuthProvider = confirm.Confirm
	hfDeleteExternalAuthProvider  = func(cmd *cobra.Command, args []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		if err := runHyperfleetDelete(cmd.Context(), r, args); err != nil {
			r.Reporter.Errorf("%v", hyperfleet.WithAPIErrorDetails(err))
			hfExitFn(1)
		}
	}
)

func runHyperfleetDelete(ctx context.Context, r *rosa.Runtime, argv []string) error {
	if len(argv) != 1 {
		return fmt.Errorf("expected exactly one command line parameter containing the name of the external authentication provider")
	}
	providerName := argv[0]
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		return err
	}
	if clusterKey == "" {
		return fmt.Errorf("--cluster is required")
	}

	cluster, err := hyperfleet.GetReadyClusterByKey(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		return err
	}
	providers := hyperfleet.ExternalAuthProviders(cluster)
	providerExists := false
	for _, provider := range providers {
		if provider.Name == providerName {
			providerExists = true
			break
		}
	}
	if !providerExists {
		return fmt.Errorf("external authentication provider '%s' not found", providerName)
	}

	if !hfConfirmExternalAuthProvider("delete external authentication provider %s on cluster %s", providerName, clusterKey) {
		return nil
	}

	patch, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"hostedCluster": map[string]any{
				"configuration": map[string]any{
					"authentication": map[string]any{
						"type":          nil,
						"oidcProviders": nil,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to encode external authentication provider deletion: %w", err)
	}

	_, err = r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Patch(
		ctx,
		string(cluster.UID),
		types.MergePatchType,
		patch,
		platform.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete external authentication provider '%s' on cluster '%s': %w",
			providerName, clusterKey, err)
	}

	r.Reporter.Infof("Successfully deleted external authentication provider '%s' from cluster '%s'",
		providerName, clusterKey)
	return nil
}
