package hyperfleet

import (
	"context"
	"fmt"
	"strings"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// ResolveClusterUID looks up a cluster by name or UID and returns its UID.
func ResolveClusterUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterKey string,
) (string, error) {
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterKey || string(c.UID) == clusterKey {
			return string(c.UID), nil
		}
	}
	return "", fmt.Errorf("cluster '%s' not found", clusterKey)
}

// GetCluster looks up a cluster by name or UID and returns the cluster object.
// Returns nil if the cluster is not found.
func GetCluster(
	ctx context.Context, client hyperfleetclientset.Interface, clusterKey string,
) (*v1alpha1.Cluster, error) {
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterKey || string(c.UID) == clusterKey {
			return &c, nil
		}
	}
	return nil, nil
}

// HasClusterUsingOperatorRolesPrefix checks if any cluster is using the given operator roles prefix.
func HasClusterUsingOperatorRolesPrefix(
	ctx context.Context, client hyperfleetclientset.Interface, prefix string,
) (bool, error) {
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, c := range list.Items {
		// Extract the prefix from the cluster's operator roles
		if c.Spec.HostedCluster.Platform.AWS != nil {
			rolesRef := c.Spec.HostedCluster.Platform.AWS.RolesRef
			// Use NodePoolManagementARN to extract the prefix
			// Format: arn:aws:iam::123456789012:role/prefix-node-pool-management
			if arn := rolesRef.NodePoolManagementARN; arn != "" {
				// Extract role name from ARN
				parts := strings.Split(arn, "/")
				if len(parts) >= 2 {
					roleName := parts[len(parts)-1]
					// Remove the suffix to get the prefix
					clusterPrefix := strings.TrimSuffix(roleName, "-"+SuffixNodePoolManagement)
					if clusterPrefix == prefix {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}
