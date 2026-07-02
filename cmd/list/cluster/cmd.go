/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	rosasdk "github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/client"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "clusters",
	Aliases: []string{"cluster"},
	Short:   "List clusters",
	Long:    "List clusters.",
	Example: `  # List all clusters
  rosa list clusters

  # List only hyperfleet clusters
  rosa list clusters --hyperfleet`,
	Args: cobra.NoArgs,
	Run:  run,
}

const clusterCount = 1000

var args struct {
	listAll        bool
	accountRoleArn string
	hyperfleet     bool
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	output.AddFlag(Cmd)
	flags.BoolVarP(&args.listAll, "all", "a", false, "List all clusters across different AWS "+
		"accounts under the same Red Hat organization")
	flags.StringVar(&args.accountRoleArn, "account-role-arn", "", "List all clusters "+
		"using the account role identified by the ARN")
	flags.BoolVar(&args.hyperfleet, "hyperfleet", false, "List only hyperfleet clusters")
}

func listClustersUsingAccountRole(creator *aws.Creator, runtime *rosa.Runtime) ([]*v1.Cluster, error) {
	role, err := runtime.AWSClient.GetAccountRoleByArn(args.accountRoleArn)
	if err != nil {
		return []*v1.Cluster{}, err
	}

	return runtime.OCMClient.GetClustersUsingAccountRole(creator, role, clusterCount, false)
}

// extractRegionFromAPIGatewayURL extracts the AWS region from an API Gateway URL
// Format: https://{api-id}.execute-api.{region}.amazonaws.com/{stage}
func extractRegionFromAPIGatewayURL(apiURL string) string {
	// Look for .execute-api.{region}.amazonaws.com pattern
	if !strings.Contains(apiURL, ".execute-api.") {
		return ""
	}

	// Split on .execute-api. and take the part after it
	parts := strings.Split(apiURL, ".execute-api.")
	if len(parts) < 2 {
		return ""
	}

	// The region is between .execute-api. and .amazonaws.com
	regionPart := parts[1]
	dotIndex := strings.Index(regionPart, ".")
	if dotIndex == -1 {
		return ""
	}

	return regionPart[:dotIndex]
}

func listHyperfleetClusters(r *rosa.Runtime) error {
	// Get base URL from environment or use default
	baseURL := os.Getenv("HYPERFLEET_API_URL")
	if baseURL == "" {
		return fmt.Errorf("HYPERFLEET_API_URL environment variable is required\n\n" +
			"Please set the hyperfleet platform API URL:\n" +
			"  export HYPERFLEET_API_URL=https://your-hyperfleet-api-gateway.execute-api.us-east-1.amazonaws.com/prod\n\n" +
			"Contact your platform administrator for the correct API URL.")
	}

	// Extract region from API Gateway URL
	// Format: https://{api-id}.execute-api.{region}.amazonaws.com/{stage}
	region := extractRegionFromAPIGatewayURL(baseURL)

	// Allow override via environment variable
	if envRegion := os.Getenv("HYPERFLEET_API_REGION"); envRegion != "" {
		r.Logger.Debugf("Overriding detected region %s with HYPERFLEET_API_REGION=%s", region, envRegion)
		region = envRegion
	}

	if region == "" {
		return fmt.Errorf("could not determine API region from URL: %s\n\n"+
			"Please set HYPERFLEET_API_REGION environment variable:\n"+
			"  export HYPERFLEET_API_REGION=us-east-1", baseURL)
	}

	r.Logger.Debugf("Creating hyperfleet client with baseURL=%s, region=%s, accountID=%s",
		baseURL, region, r.Creator.AccountID)

	// Create hyperfleet SDK client with AWS SigV4 authentication
	client, err := rosasdk.NewClient(
		rosasdk.WithBaseURL(baseURL),
		rosasdk.WithRegion(region),
		rosasdk.WithUserAgent("rosa-cli"),
		rosasdk.WithAccountID(r.Creator.AccountID),
	)
	if err != nil {
		return fmt.Errorf("failed to create hyperfleet client: %w", err)
	}

	r.Logger.Debugf("Hyperfleet client created successfully")

	// List clusters from hyperfleet platform API
	r.Logger.Debugf("Calling ListClusters API")
	result, err := client.ListClusters(context.Background(), nil)
	if err != nil {
		r.Logger.Errorf("ListClusters API call failed: %v", err)
		return fmt.Errorf("failed to list hyperfleet clusters: %w\n\n"+
			"Troubleshooting:\n"+
			"  - Ensure HYPERFLEET_API_URL is set correctly (current: %s)\n"+
			"  - Verify your AWS credentials have permissions to access the hyperfleet API\n"+
			"  - Check that your AWS account (%s) is authorized for the hyperfleet platform\n"+
			"  - Verify the API region matches your AWS region (%s)",
			err, baseURL, r.Creator.AccountID, region)
	}
	r.Logger.Debugf("Retrieved %d clusters", result.Total)

	// Handle output format
	if output.HasFlag() {
		err = output.Print(result.Items)
		if err != nil {
			return fmt.Errorf("failed to print output: %w", err)
		}
		return nil
	}

	// Display clusters in table format
	if !output.HasFlag() {
		r.Reporter.Infof("Hyperfleet API: %s (region: %s)", baseURL, region)
		fmt.Println()
	}

	if len(result.Items) == 0 {
		r.Reporter.Infof("No hyperfleet clusters available")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tCREATED BY\tCREATED AT\n")
	for _, cluster := range result.Items {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			cluster.Id,
			cluster.Name,
			cluster.CreatedBy,
			cluster.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	writer.Flush()

	return nil
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWSWarnInsteadOfExit().WithOCM()
	defer r.Cleanup()

	// If --hyperfleet is specified, use the hyperfleet platform API
	if args.hyperfleet {
		err := listHyperfleetClusters(r)
		if err != nil {
			r.Reporter.Errorf("Failed to get hyperfleet clusters: %v", err)
			os.Exit(1)
		}
		return
	}

	// Retrieve the list of clusters:
	var creator *aws.Creator
	if args.listAll {
		creator = nil
	} else {
		creator = r.Creator
	}

	var clusters []*v1.Cluster
	var err error

	if args.accountRoleArn != "" {
		clusters, err = listClustersUsingAccountRole(creator, r)
	} else {
		clusters, err = r.OCMClient.GetClusters(creator, clusterCount, false)
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get clusters: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(clusters)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(clusters) == 0 {
		r.Reporter.Infof("No clusters available")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tSTATE\tTOPOLOGY\n")
	for _, cluster := range clusters {
		typeOutput := "Classic"
		if cluster.AWS() != nil && cluster.AWS().STS() != nil && cluster.AWS().STS().Enabled() {
			typeOutput = "Classic (STS)"
		}
		if cluster.Hypershift().Enabled() {
			typeOutput = "Hosted CP"
		}
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			cluster.ID(),
			cluster.Name(),
			cluster.State(),
			typeOutput,
		)
	}
	writer.Flush()
}
