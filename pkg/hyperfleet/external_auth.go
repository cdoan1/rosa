package hyperfleet

import (
	"context"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	configv1 "github.com/openshift/api/config/v1"
)

// GetReadyClusterByKey resolves a Platform API cluster by name or UID, fetches
// its complete HostedCluster spec, and requires it to be ready for day-two
// operations.
func GetReadyClusterByKey(
	ctx context.Context, client hyperfleetclientset.Interface, clusterKey string,
) (*v1alpha1.Cluster, error) {
	cluster, err := GetCluster(ctx, client, clusterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve cluster '%s': %w", clusterKey, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("cluster '%s' not found", clusterKey)
	}
	if cluster.UID == "" {
		return nil, fmt.Errorf("cluster '%s' did not include a Platform API UID", clusterKey)
	}

	cluster, err = client.HyperfleetV1alpha1().Clusters().Get(ctx, string(cluster.UID), platform.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster '%s': %w", clusterKey, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("cluster '%s' not found", clusterKey)
	}
	if cluster.Status.Phase != v1alpha1.ClusterPhaseReady {
		return nil, fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}
	return cluster, nil
}

// ExternalAuthProviders returns the OIDC providers configured on a Platform
// API cluster. It always returns a non-nil slice.
func ExternalAuthProviders(cluster *v1alpha1.Cluster) []configv1.OIDCProvider {
	if cluster == nil || cluster.Spec.HostedCluster.Configuration == nil ||
		cluster.Spec.HostedCluster.Configuration.Authentication == nil {
		return []configv1.OIDCProvider{}
	}
	providers := cluster.Spec.HostedCluster.Configuration.Authentication.OIDCProviders
	if providers == nil {
		return []configv1.OIDCProvider{}
	}
	return providers
}
