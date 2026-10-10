// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfExitFn = func(code int) { os.Exit(code) }

	hfCreateExternalAuthProvider = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		if err := runHyperfleetExternalAuthProvider(cmd.Context(), r, cmd); err != nil {
			r.Reporter.Errorf("%v", hyperfleet.WithAPIErrorDetails(err))
			hfExitFn(1)
		}
	}
)

type hyperfleetExternalAuthInput struct {
	name              string
	issuerURL         string
	issuerAudiences   []string
	issuerCAName      string
	discoveryURL      string
	usernameClaim     string
	groupsClaim       string
	groupsClaimPrefix string
	claimRules        []map[string]any
}

func rejectHyperfleetOnlyFlagsV1(cmd *cobra.Command) error {
	for _, name := range []string{"issuer-ca-name", "discovery-url", "claim-mapping-groups-prefix"} {
		if flag := cmd.Flags().Lookup(name); flag != nil && flag.Changed {
			return fmt.Errorf("--%s is only supported with Platform API v2", name)
		}
	}
	return nil
}

func runHyperfleetExternalAuthProvider(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command) error {
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		return err
	}
	if clusterKey == "" {
		return fmt.Errorf("--cluster is required")
	}

	cluster, err := hyperfleet.GetCluster(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		return fmt.Errorf("failed to resolve cluster '%s': %w", clusterKey, err)
	}
	if cluster == nil {
		return fmt.Errorf("cluster '%s' not found", clusterKey)
	}
	if cluster.Status.Phase != v1alpha1.ClusterPhaseReady {
		return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
	}
	if cluster.UID == "" {
		return fmt.Errorf("cluster '%s' did not include a Platform API UID", clusterKey)
	}
	if configuration := cluster.Spec.HostedCluster.Configuration; configuration != nil {
		if authentication := configuration.Authentication; authentication != nil && len(authentication.OIDCProviders) > 0 {
			return fmt.Errorf("cluster '%s' already has an external authentication provider configured", clusterKey)
		}
	}
	if err := rejectUnsupportedHyperfleetProviderFlags(cmd); err != nil {
		return err
	}

	input, err := resolveHyperfleetExternalAuthInput(cmd, r)
	if err != nil {
		return err
	}
	patch, err := buildHyperfleetExternalAuthPatch(input)
	if err != nil {
		return err
	}

	_, err = r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Patch(
		ctx,
		string(cluster.UID),
		types.MergePatchType,
		patch,
		platform.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create external authentication provider for cluster '%s': %w", clusterKey, err)
	}

	r.Reporter.Infof("Created external authentication provider '%s' for cluster '%s'", input.name, clusterKey)
	return nil
}

func rejectUnsupportedHyperfleetProviderFlags(cmd *cobra.Command) error {
	unsupported := []struct {
		flag   string
		reason string
	}{
		{
			flag:   "issuer-ca-file",
			reason: "use --issuer-ca-name to reference a ConfigMap in openshift-config",
		},
		{
			flag:   "console-client-id",
			reason: "console OIDC clients are not supported by this command in Platform API v2",
		},
		{
			flag:   "console-client-secret",
			reason: "console OIDC clients are not supported by this command in Platform API v2",
		},
	}
	for _, item := range unsupported {
		if flag := cmd.Flags().Lookup(item.flag); flag != nil && flag.Changed {
			return fmt.Errorf("--%s is not supported with Platform API v2: %s", item.flag, item.reason)
		}
	}
	return nil
}

func resolveHyperfleetExternalAuthInput(cmd *cobra.Command, r *rosa.Runtime) (*hyperfleetExternalAuthInput, error) {
	input := &hyperfleetExternalAuthInput{}
	var err error

	input.name, _ = cmd.Flags().GetString("name")
	input.issuerURL, _ = cmd.Flags().GetString("issuer-url")
	input.issuerAudiences, _ = cmd.Flags().GetStringSlice("issuer-audiences")
	input.usernameClaim, _ = cmd.Flags().GetString("claim-mapping-username-claim")
	input.groupsClaim, _ = cmd.Flags().GetString("claim-mapping-groups-claim")
	input.issuerCAName = hfExternalAuthFlags.issuerCAName
	input.discoveryURL = hfExternalAuthFlags.discoveryURL
	input.groupsClaimPrefix = hfExternalAuthFlags.groupsClaimPrefix

	if !hasRequiredHyperfleetExternalAuthInput(input) && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	if strings.TrimSpace(input.name) == "" {
		input.name, err = promptHyperfleetExternalAuthValue(cmd, "name", "Name", input.name, true, nil)
		if err != nil {
			return nil, err
		}
	}
	if len(input.issuerAudiences) == 0 {
		audiences, promptErr := promptHyperfleetExternalAuthValue(
			cmd, "issuer-audiences", "Issuer audiences", "", true, nil,
		)
		if promptErr != nil {
			return nil, promptErr
		}
		input.issuerAudiences = strings.Split(audiences, ",")
	}
	if strings.TrimSpace(input.issuerURL) == "" {
		input.issuerURL, err = promptHyperfleetExternalAuthValue(
			cmd, "issuer-url", "The serving URL of the token issuer", input.issuerURL,
			true, []interactive.Validator{interactive.IsURL},
		)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(input.usernameClaim) == "" {
		input.usernameClaim, err = promptHyperfleetExternalAuthValue(
			cmd, "claim-mapping-username-claim", "Claim mapping username", "email", true, nil,
		)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(input.groupsClaim) == "" {
		input.groupsClaim, err = promptHyperfleetExternalAuthValue(
			cmd, "claim-mapping-groups-claim", "Claim mapping groups", "groups", true, nil,
		)
		if err != nil {
			return nil, err
		}
	}

	if interactive.Enabled() {
		if !cmd.Flags().Changed("issuer-ca-name") {
			input.issuerCAName, err = promptHyperfleetExternalAuthValue(
				cmd, "issuer-ca-name", "Issuer CA ConfigMap name", input.issuerCAName, false, nil,
			)
			if err != nil {
				return nil, err
			}
		}
		if !cmd.Flags().Changed("discovery-url") {
			input.discoveryURL, err = promptHyperfleetExternalAuthValue(
				cmd, "discovery-url", "OIDC discovery URL", input.discoveryURL,
				false, []interactive.Validator{interactive.IsURL},
			)
			if err != nil {
				return nil, err
			}
		}
		if !cmd.Flags().Changed("claim-mapping-groups-prefix") {
			input.groupsClaimPrefix, err = promptHyperfleetExternalAuthValue(
				cmd, "claim-mapping-groups-prefix", "Groups claim prefix", input.groupsClaimPrefix, false, nil,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	input.name = strings.TrimSpace(input.name)
	input.issuerURL = strings.TrimSpace(input.issuerURL)
	input.usernameClaim = strings.TrimSpace(input.usernameClaim)
	input.groupsClaim = strings.TrimSpace(input.groupsClaim)
	input.issuerCAName = strings.TrimSpace(input.issuerCAName)
	input.discoveryURL = strings.TrimSpace(input.discoveryURL)
	input.groupsClaimPrefix = strings.TrimSpace(input.groupsClaimPrefix)

	if input.name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	if err := validateHyperfleetIssuerURL("issuer URL", input.issuerURL); err != nil {
		return nil, err
	}
	input.issuerAudiences = normalizeHyperfleetAudiences(input.issuerAudiences)
	if len(input.issuerAudiences) == 0 {
		return nil, fmt.Errorf("--issuer-audiences must include at least one audience")
	}
	if len(input.issuerAudiences) > 10 {
		return nil, fmt.Errorf("--issuer-audiences supports at most 10 audiences")
	}
	if input.usernameClaim == "" {
		return nil, fmt.Errorf("--claim-mapping-username-claim is required")
	}
	if input.groupsClaim == "" {
		return nil, fmt.Errorf("--claim-mapping-groups-claim is required")
	}
	if input.discoveryURL != "" {
		if err := validateHyperfleetIssuerURL("discovery URL", input.discoveryURL); err != nil {
			return nil, err
		}
		if strings.TrimRight(input.discoveryURL, "/") == strings.TrimRight(input.issuerURL, "/") {
			return nil, fmt.Errorf("--discovery-url must differ from --issuer-url")
		}
	}

	input.claimRules, err = parseHyperfleetClaimValidationRules(cmd)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func hasRequiredHyperfleetExternalAuthInput(input *hyperfleetExternalAuthInput) bool {
	return strings.TrimSpace(input.name) != "" && strings.TrimSpace(input.issuerURL) != "" &&
		len(input.issuerAudiences) > 0 && strings.TrimSpace(input.usernameClaim) != "" &&
		strings.TrimSpace(input.groupsClaim) != ""
}

func promptHyperfleetExternalAuthValue(
	cmd *cobra.Command,
	flagName string,
	question string,
	defaultValue string,
	required bool,
	validators []interactive.Validator,
) (string, error) {
	flag := cmd.Flags().Lookup(flagName)
	if flag == nil {
		return "", fmt.Errorf("flag --%s is not registered", flagName)
	}
	return interactive.GetString(interactive.Input{
		Question:   question,
		Default:    defaultValue,
		Help:       flag.Usage,
		Required:   required,
		Validators: validators,
	})
}

func validateHyperfleetIssuerURL(label, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be a valid HTTPS URL without query or fragment components", label)
	}
	return nil
}

func normalizeHyperfleetAudiences(audiences []string) []string {
	result := make([]string, 0, len(audiences))
	seen := make(map[string]struct{}, len(audiences))
	for _, audience := range audiences {
		audience = strings.TrimSpace(audience)
		if audience == "" {
			continue
		}
		if _, ok := seen[audience]; ok {
			continue
		}
		seen[audience] = struct{}{}
		result = append(result, audience)
	}
	return result
}

func parseHyperfleetClaimValidationRules(cmd *cobra.Command) ([]map[string]any, error) {
	rules, err := cmd.Flags().GetStringSlice("claim-validation-rule")
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		claim, requiredValue, found := strings.Cut(strings.TrimSpace(rule), ":")
		claim = strings.TrimSpace(claim)
		requiredValue = strings.TrimSpace(requiredValue)
		if !found || claim == "" || requiredValue == "" {
			return nil, fmt.Errorf("invalid --claim-validation-rule %q; expected <claim>:<required_value>", rule)
		}
		result = append(result, map[string]any{
			"type": "RequiredClaim",
			"requiredClaim": map[string]string{
				"claim":         claim,
				"requiredValue": requiredValue,
			},
		})
	}
	return result, nil
}

func buildHyperfleetExternalAuthPatch(input *hyperfleetExternalAuthInput) ([]byte, error) {
	issuer := map[string]any{
		"issuerURL": input.issuerURL,
		"audiences": input.issuerAudiences,
	}
	if input.issuerCAName != "" {
		issuer["issuerCertificateAuthority"] = map[string]string{"name": input.issuerCAName}
	}
	if input.discoveryURL != "" {
		issuer["discoveryURL"] = input.discoveryURL
	}

	groupsClaim := map[string]any{"claim": input.groupsClaim}
	if input.groupsClaimPrefix != "" {
		groupsClaim["prefix"] = input.groupsClaimPrefix
	}

	provider := map[string]any{
		"name":   input.name,
		"issuer": issuer,
		"claimMappings": map[string]any{
			"username": map[string]any{
				"claim":        input.usernameClaim,
				"prefixPolicy": "NoPrefix",
			},
			"groups": groupsClaim,
		},
	}
	if len(input.claimRules) > 0 {
		provider["claimValidationRules"] = input.claimRules
	}

	patch := map[string]any{
		"spec": map[string]any{
			"hostedCluster": map[string]any{
				"configuration": map[string]any{
					"authentication": map[string]any{
						"type":          "OIDC",
						"oidcProviders": []map[string]any{provider},
					},
				},
			},
		},
	}
	encoded, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to encode external authentication provider: %w", err)
	}
	return encoded, nil
}
