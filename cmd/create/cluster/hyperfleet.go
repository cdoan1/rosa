package cluster

import (
	"context"
	"net"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/api/util/ipnet"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, hfDescribeSubnets, and hfCreateCluster are package-level
// vars so tests can stub the hyperfleet dispatch path without real AWS calls.
var (
	hfEnabled         = hyperfleet.Enabled
	hfExitFn          = func(code int) { os.Exit(code) }
	hfDescribeSubnets = func(
		ctx context.Context, cfg awssdk.Config, subnetID string,
	) (*ec2svc.DescribeSubnetsOutput, error) {
		return ec2svc.NewFromConfig(cfg).DescribeSubnets(ctx, &ec2svc.DescribeSubnetsInput{
			SubnetIds: []string{subnetID},
		})
	}
	hfCreateCluster = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleet(r)
	}
)

// runHyperfleet creates an HCP cluster via the Platform API v2.
// It derives VPC ID and availability zone from the provided subnet.
func runHyperfleet(r *rosa.Runtime) {
	ctx := context.Background()

	clusterName := args.clusterName
	if clusterName == "" {
		r.Reporter.Errorf("--cluster-name is required")
		hfExitFn(1)
		return
	}

	if args.operatorRolesPrefix == "" {
		r.Reporter.Errorf("--operator-roles-prefix is required")
		hfExitFn(1)
		return
	}

	if len(args.subnetIDs) == 0 {
		r.Reporter.Errorf("--subnet-ids is required")
		hfExitFn(1)
		return
	}
	subnetID := args.subnetIDs[0]

	// Derive VPC ID and availability zone from the subnet.
	subnetOut, err := hfDescribeSubnets(ctx, r.AWSConfig, subnetID)
	if err != nil {
		r.Reporter.Errorf("Failed to describe subnet '%s': %v", subnetID, err)
		hfExitFn(1)
		return
	}
	if len(subnetOut.Subnets) == 0 {
		r.Reporter.Errorf("Subnet '%s' not found", subnetID)
		hfExitFn(1)
		return
	}
	vpcID := awssdk.ToString(subnetOut.Subnets[0].VpcId)
	if vpcID == "" {
		r.Reporter.Errorf("Subnet '%s' has no VPC ID", subnetID)
		hfExitFn(1)
		return
	}
	zone := awssdk.ToString(subnetOut.Subnets[0].AvailabilityZone)
	if zone == "" {
		r.Reporter.Errorf("Subnet '%s' has no availability zone", subnetID)
		hfExitFn(1)
		return
	}

	rolesRef := hyperfleet.ComputeRolesRef(args.operatorRolesPrefix, r.Creator.AccountID, r.Creator.Partition)

	// Build networking configuration
	networking := buildNetworkingConfig()

	subnetRef := subnetID
	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Create(
		ctx,
		&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: clusterName},
			Spec: v1alpha1.ClusterSpec{
				HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					Release: hypershiftv1beta1.Release{Image: args.version},
					Platform: hypershiftv1beta1.PlatformSpec{
						Type: hypershiftv1beta1.AWSPlatform,
						AWS: &hypershiftv1beta1.AWSPlatformSpec{
							Region:   r.Region,
							RolesRef: rolesRef,
							CloudProviderConfig: &hypershiftv1beta1.AWSCloudProviderConfig{
								VPC:  vpcID,
								Zone: zone,
								Subnet: &hypershiftv1beta1.AWSResourceReference{
									ID: &subnetRef,
								},
							},
						},
					},
					Networking: networking,
				},
			},
		},
		platform.CreateOptions{},
	)
	if err != nil {
		r.Reporter.Errorf("Failed to create cluster '%s': %v", clusterName, err)
		hfExitFn(1)
		return
	}

	r.Reporter.Infof("Cluster '%s' created with ID '%s'", clusterName, string(cluster.UID))
}

// buildNetworkingConfig constructs the networking configuration from CLI args,
// applying defaults that match the hyperfleet-operator behavior when values are not specified.
func buildNetworkingConfig() hypershiftv1beta1.ClusterNetworking {
	// Default values matching hyperfleet-operator defaults
	const (
		defaultNetworkType   = "OVNKubernetes"
		defaultServiceCIDR   = "172.31.0.0/16"
		defaultMachineCIDR   = "10.0.0.0/16"
		defaultPodCIDR       = "10.132.0.0/14"
		defaultHostPrefix    = 23
	)

	networking := hypershiftv1beta1.ClusterNetworking{}

	// Network Type
	if args.networkType != "" {
		networking.NetworkType = hypershiftv1beta1.NetworkType(args.networkType)
	} else {
		networking.NetworkType = hypershiftv1beta1.NetworkType(defaultNetworkType)
	}

	// Service CIDR
	serviceCIDRNet := parseOrDefault(args.serviceCIDR, defaultServiceCIDR)
	networking.ServiceNetwork = []hypershiftv1beta1.ServiceNetworkEntry{
		{CIDR: *ipnet.MustParseCIDR(serviceCIDRNet)},
	}

	// Machine CIDR
	machineCIDRNet := parseOrDefault(args.machineCIDR, defaultMachineCIDR)
	networking.MachineNetwork = []hypershiftv1beta1.MachineNetworkEntry{
		{CIDR: *ipnet.MustParseCIDR(machineCIDRNet)},
	}

	// Pod CIDR and Host Prefix
	podCIDRNet := parseOrDefault(args.podCIDR, defaultPodCIDR)
	hostPrefix := int32(defaultHostPrefix)
	if args.hostPrefix > 0 {
		hostPrefix = int32(args.hostPrefix)
	}
	networking.ClusterNetwork = []hypershiftv1beta1.ClusterNetworkEntry{
		{
			CIDR:       *ipnet.MustParseCIDR(podCIDRNet),
			HostPrefix: hostPrefix,
		},
	}

	return networking
}

// parseOrDefault returns the string representation of the IPNet if it's set,
// otherwise returns the default string.
func parseOrDefault(ipNet net.IPNet, defaultCIDR string) string {
	if ipNet.IP != nil {
		return ipNet.String()
	}
	return defaultCIDR
}
