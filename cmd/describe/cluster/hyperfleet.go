package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled         = hyperfleet.Enabled
	exitFn            = func(code int) { os.Exit(code) }
	hfDescribeCluster = func(cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDescribe(r, cmd, argv)
	}
)

func runHyperfleetDescribe(r *rosa.Runtime, cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters()
	cluster, err := clusters.Get(ctx, clusterID, platform.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	// Debug: Uncomment to check what we're receiving from the API
	// fmt.Fprintf(os.Stderr, "DEBUG: NetworkType: '%s'\n", cluster.Spec.HostedCluster.Networking.NetworkType)
	// fmt.Fprintf(os.Stderr, "DEBUG: ServiceNetwork count: %d\n", len(cluster.Spec.HostedCluster.Networking.ServiceNetwork))
	// fmt.Fprintf(os.Stderr, "DEBUG: MachineNetwork count: %d\n", len(cluster.Spec.HostedCluster.Networking.MachineNetwork))
	// fmt.Fprintf(os.Stderr, "DEBUG: ClusterNetwork count: %d\n", len(cluster.Spec.HostedCluster.Networking.ClusterNetwork))

	if output.HasFlag() {
		m := hfClusterToMap(cluster)
		if err := output.Print(m); err != nil {
			r.Reporter.Errorf("%s", err)
			exitFn(1)
		}
		return
	}

	fmt.Print(hfClusterToString(cluster))
}

// hfClusterToMap converts a hyperfleet Cluster to a generic map suitable for
// JSON/YAML structured output, mirroring the shape of formatClusterHypershift.
func hfClusterToMap(c *v1alpha1.Cluster) map[string]interface{} {
	aws := c.Spec.HostedCluster.Platform.AWS

	rolesRef := map[string]string{}
	if aws != nil {
		ref := aws.RolesRef
		rolesRef["ingressARN"] = ref.IngressARN
		rolesRef["imageRegistryARN"] = ref.ImageRegistryARN
		rolesRef["storageARN"] = ref.StorageARN
		rolesRef["networkARN"] = ref.NetworkARN
		rolesRef["kubeCloudControllerARN"] = ref.KubeCloudControllerARN
		rolesRef["controlPlaneOperatorARN"] = ref.ControlPlaneOperatorARN
		rolesRef["nodePoolManagementARN"] = ref.NodePoolManagementARN
	}

	spec := map[string]interface{}{
		"oidc_issuer": c.Spec.HostedCluster.IssuerURL,
		"roles_ref":   rolesRef,
		// TODO: Uncomment when platform-api exposes these fields
		// "controllerAvailabilityPolicy":       string(c.Spec.HostedCluster.ControllerAvailabilityPolicy),
		// "infrastructureAvailabilityPolicy":   string(c.Spec.HostedCluster.InfrastructureAvailabilityPolicy),
	}

	m := map[string]interface{}{
		"id":            string(c.UID),
		"name":          c.Name,
		"control_plane": "ROSA Service Hosted",
		"state":         string(c.Status.Phase),
		"created_at":    c.CreationTimestamp.UTC().Format(time.RFC3339),
		"spec":          spec,
	}

	if aws != nil {
		m["region"] = aws.Region
		if aws.CloudProviderConfig != nil {
			m["vpc"] = aws.CloudProviderConfig.VPC
			if aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
				m["subnet"] = *aws.CloudProviderConfig.Subnet.ID
			}
		}
	}

	if c.Status.Version != "" {
		m["version"] = c.Status.Version
	}
	if c.Status.ControlPlaneEndpoint.Host != "" {
		m["api_url"] = fmt.Sprintf("https://%s:%d",
			c.Status.ControlPlaneEndpoint.Host,
			c.Status.ControlPlaneEndpoint.Port)
	}
	if c.Status.PlacementRef != nil {
		m["management_cluster"] = c.Status.PlacementRef.ManagementCluster
	}
	if c.Spec.ExpirationTimestamp != nil {
		m["expiration"] = c.Spec.ExpirationTimestamp.UTC().Format(time.RFC3339)
	}

	// Add DNS information
	dns := make(map[string]interface{})
	if c.Status.ControlPlaneEndpoint.Host != "" {
		dns["api_endpoint"] = c.Status.ControlPlaneEndpoint.Host
		// Extract base domain from the API endpoint if possible
		// API endpoint typically looks like: api.<cluster-name>.<base-domain>
		parts := strings.SplitN(c.Status.ControlPlaneEndpoint.Host, ".", 3)
		if len(parts) >= 3 {
			dns["cluster_domain"] = strings.Join(parts[1:], ".")
			dns["base_domain"] = parts[2]
		}
	}
	if len(dns) > 0 {
		m["dns"] = dns
	}

	// Add Network information
	network := make(map[string]interface{})
	if c.Spec.HostedCluster.Networking.NetworkType != "" {
		network["type"] = string(c.Spec.HostedCluster.Networking.NetworkType)
	}

	// Service CIDR
	if len(c.Spec.HostedCluster.Networking.ServiceNetwork) > 0 {
		serviceCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.ServiceNetwork))
		for _, sn := range c.Spec.HostedCluster.Networking.ServiceNetwork {
			serviceCIDRs = append(serviceCIDRs, sn.CIDR.String())
		}
		network["service_cidr"] = strings.Join(serviceCIDRs, ", ")
	}

	// Machine CIDR
	if len(c.Spec.HostedCluster.Networking.MachineNetwork) > 0 {
		machineCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.MachineNetwork))
		for _, mn := range c.Spec.HostedCluster.Networking.MachineNetwork {
			machineCIDRs = append(machineCIDRs, mn.CIDR.String())
		}
		network["machine_cidr"] = strings.Join(machineCIDRs, ", ")
	}

	// Pod CIDR (ClusterNetwork)
	if len(c.Spec.HostedCluster.Networking.ClusterNetwork) > 0 {
		podCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.ClusterNetwork))
		for _, cn := range c.Spec.HostedCluster.Networking.ClusterNetwork {
			podCIDRs = append(podCIDRs, cn.CIDR.String())
		}
		network["pod_cidr"] = strings.Join(podCIDRs, ", ")
		if len(c.Spec.HostedCluster.Networking.ClusterNetwork) > 0 && c.Spec.HostedCluster.Networking.ClusterNetwork[0].HostPrefix != 0 {
			network["host_prefix"] = c.Spec.HostedCluster.Networking.ClusterNetwork[0].HostPrefix
		}
	}

	// Subnets
	if aws != nil && aws.CloudProviderConfig != nil && aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
		network["subnets"] = []string{*aws.CloudProviderConfig.Subnet.ID}
	}

	if len(network) > 0 {
		m["network"] = network
	}

	conditions := make([]map[string]interface{}, 0, len(c.Status.Conditions))
	for _, cond := range c.Status.Conditions {
		conditions = append(conditions, map[string]interface{}{
			"type":    cond.Type,
			"status":  string(cond.Status),
			"reason":  cond.Reason,
			"message": cond.Message,
		})
	}
	m["conditions"] = conditions

	return m
}

// hfClusterToString formats a hyperfleet Cluster as a human-readable string,
// following the same label-alignment style as rosa describe cluster.
func hfClusterToString(c *v1alpha1.Cluster) string {
	aws := c.Spec.HostedCluster.Platform.AWS

	region := ""
	vpc := ""
	subnet := ""
	if aws != nil {
		region = aws.Region
		if aws.CloudProviderConfig != nil {
			vpc = aws.CloudProviderConfig.VPC
			if aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
				subnet = *aws.CloudProviderConfig.Subnet.ID
			}
		}
	}

	apiURL := ""
	if c.Status.ControlPlaneEndpoint.Host != "" {
		apiURL = fmt.Sprintf("https://%s:%d",
			c.Status.ControlPlaneEndpoint.Host,
			c.Status.ControlPlaneEndpoint.Port)
	}

	s := fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"Control Plane:              %s\n"+
		"OpenShift Version:          %s\n"+
		"API URL:                    %s\n"+
		"Region:                     %s\n"+
		"VPC:                        %s\n"+
		"Subnet:                     %s\n"+
		"OIDC Endpoint URL:          %s\n"+
		"State:                      %s\n"+
		"Created:                    %s\n",
		c.Name,
		string(c.UID),
		"ROSA Service Hosted",
		c.Status.Version,
		apiURL,
		region,
		vpc,
		subnet,
		c.Spec.HostedCluster.IssuerURL,
		string(c.Status.Phase),
		c.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"),
	)

	if c.Status.PlacementRef != nil {
		s += fmt.Sprintf("Management Cluster:         %s\n", c.Status.PlacementRef.ManagementCluster)
	}

	// TODO: Uncomment when platform-api exposes these fields
	// Add availability policies if present
	// if c.Spec.HostedCluster.ControllerAvailabilityPolicy != "" {
	// 	s += fmt.Sprintf("Controller Availability:    %s\n",
	// 		string(c.Spec.HostedCluster.ControllerAvailabilityPolicy))
	// }
	// if c.Spec.HostedCluster.InfrastructureAvailabilityPolicy != "" {
	// 	s += fmt.Sprintf("Infrastructure Availability: %s\n",
	// 		string(c.Spec.HostedCluster.InfrastructureAvailabilityPolicy))
	// }

	if c.Spec.ExpirationTimestamp != nil {
		s += fmt.Sprintf("Expiration:                 %s\n",
			c.Spec.ExpirationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	// Add Network information
	s += "Network:\n"
	if c.Spec.HostedCluster.Networking.NetworkType != "" {
		s += fmt.Sprintf(" - Type:                    %s\n", string(c.Spec.HostedCluster.Networking.NetworkType))
	}
	if len(c.Spec.HostedCluster.Networking.ServiceNetwork) > 0 {
		serviceCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.ServiceNetwork))
		for _, sn := range c.Spec.HostedCluster.Networking.ServiceNetwork {
			serviceCIDRs = append(serviceCIDRs, sn.CIDR.String())
		}
		s += fmt.Sprintf(" - Service CIDR:            %s\n", strings.Join(serviceCIDRs, ", "))
	}
	if len(c.Spec.HostedCluster.Networking.MachineNetwork) > 0 {
		machineCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.MachineNetwork))
		for _, mn := range c.Spec.HostedCluster.Networking.MachineNetwork {
			machineCIDRs = append(machineCIDRs, mn.CIDR.String())
		}
		s += fmt.Sprintf(" - Machine CIDR:            %s\n", strings.Join(machineCIDRs, ", "))
	}
	if len(c.Spec.HostedCluster.Networking.ClusterNetwork) > 0 {
		podCIDRs := make([]string, 0, len(c.Spec.HostedCluster.Networking.ClusterNetwork))
		for _, cn := range c.Spec.HostedCluster.Networking.ClusterNetwork {
			podCIDRs = append(podCIDRs, cn.CIDR.String())
		}
		s += fmt.Sprintf(" - Pod CIDR:                %s\n", strings.Join(podCIDRs, ", "))
		if c.Spec.HostedCluster.Networking.ClusterNetwork[0].HostPrefix != 0 {
			s += fmt.Sprintf(" - Host Prefix:             /%d\n", c.Spec.HostedCluster.Networking.ClusterNetwork[0].HostPrefix)
		}
	}
	if aws != nil && aws.CloudProviderConfig != nil && aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
		s += fmt.Sprintf(" - Subnets:                 %s\n", *aws.CloudProviderConfig.Subnet.ID)
	}

	if aws != nil {
		var roles []string
		for _, arn := range []string{
			aws.RolesRef.IngressARN,
			aws.RolesRef.ImageRegistryARN,
			aws.RolesRef.StorageARN,
			aws.RolesRef.NetworkARN,
			aws.RolesRef.KubeCloudControllerARN,
			aws.RolesRef.ControlPlaneOperatorARN,
			aws.RolesRef.NodePoolManagementARN,
		} {
			if arn != "" {
				roles = append(roles, arn)
			}
		}
		if len(roles) > 0 {
			s += "Operator IAM Roles:\n"
			for _, arn := range roles {
				s += fmt.Sprintf(" - %s\n", arn)
			}
		}
	}

	if len(c.Status.Conditions) > 0 {
		s += "Conditions:\n"
		for _, cond := range c.Status.Conditions {
			s += fmt.Sprintf(" - %-10s %-5s  %s\n",
				cond.Type+":", string(cond.Status),
				conditionSummary(cond.Reason, cond.Message))
		}
	}

	return s
}

// conditionSummary returns a concise reason+message string for a condition row.
func conditionSummary(reason, message string) string {
	if reason == "" {
		return message
	}
	if message == "" || strings.HasPrefix(message, reason) {
		return reason
	}
	return reason + ": " + message
}
