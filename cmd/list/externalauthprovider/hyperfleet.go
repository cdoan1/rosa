// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfExitFn                    = func(code int) { os.Exit(code) }
	hfListExternalAuthProviders = func(cmd *cobra.Command, _ []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		if err := runHyperfleetList(cmd.Context(), r, cmd); err != nil {
			r.Reporter.Errorf("%v", hyperfleet.WithAPIErrorDetails(err))
			hfExitFn(1)
		}
	}
)

func runHyperfleetList(ctx context.Context, r *rosa.Runtime, _ *cobra.Command) error {
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

	if output.HasFlag() {
		if err := output.Print(providers); err != nil {
			return fmt.Errorf("failed to print external authentication providers: %w", err)
		}
		return nil
	}

	if len(providers) == 0 {
		r.Reporter.Infof("There are no external authentication providers for cluster '%s'", clusterKey)
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "NAME\tISSUER URL")
	for _, provider := range providers {
		_, _ = fmt.Fprintf(writer, "%s\t%s\n", provider.Name, provider.Issuer.URL)
	}
	return writer.Flush()
}
