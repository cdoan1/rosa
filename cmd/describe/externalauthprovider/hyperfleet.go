// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"context"
	"fmt"
	"os"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfExitFn                       = func(code int) { os.Exit(code) }
	hfDescribeExternalAuthProvider = func(cmd *cobra.Command, args []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		if err := runHyperfleetDescribe(cmd.Context(), r, cmd, args); err != nil {
			r.Reporter.Errorf("%v", hyperfleet.WithAPIErrorDetails(err))
			hfExitFn(1)
		}
	}
)

func runHyperfleetDescribe(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	providerName, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	if len(argv) == 1 && !cmd.Flag("name").Changed {
		providerName = argv[0]
	}
	if providerName == "" {
		return fmt.Errorf("you need to specify an external authentication provider name with '--name' parameter")
	}

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

	var provider *configv1.OIDCProvider
	providers := hyperfleet.ExternalAuthProviders(cluster)
	for i := range providers {
		candidate := &providers[i]
		if candidate.Name == providerName {
			provider = candidate
			break
		}
	}
	if provider == nil {
		return fmt.Errorf("external authentication provider '%s' not found", providerName)
	}

	if output.HasFlag() {
		if err := output.Print(provider); err != nil {
			return fmt.Errorf("failed to print external authentication provider: %w", err)
		}
		return nil
	}

	_, err = fmt.Fprint(os.Stdout, formatHyperfleetExternalAuthProvider(string(cluster.UID), provider))
	return err
}

func formatHyperfleetExternalAuthProvider(clusterID string, provider *configv1.OIDCProvider) string {
	var audiences strings.Builder
	for _, audience := range provider.Issuer.Audiences {
		fmt.Fprintf(&audiences, "\n                                       - %s", audience)
	}

	return fmt.Sprintf("\n"+
		"ID:                                    %s\n"+
		"Cluster ID:                            %s\n"+
		"Issuer audiences:                      %s\n"+
		"Issuer Url:                            %s\n"+
		"Issuer CA ConfigMap:                   %s\n"+
		"Discovery URL:                         %s\n"+
		"Claim mappings group:                  %s\n"+
		"Claim mappings groups prefix:          %s\n"+
		"Claim mappings username:               %s\n"+
		"Claim mappings username prefix policy:  %s\n",
		provider.Name,
		clusterID,
		audiences.String(),
		provider.Issuer.URL,
		provider.Issuer.CertificateAuthority.Name,
		provider.Issuer.DiscoveryURL,
		provider.ClaimMappings.Groups.Claim,
		provider.ClaimMappings.Groups.Prefix,
		provider.ClaimMappings.Username.Claim,
		provider.ClaimMappings.Username.PrefixPolicy,
	)
}
