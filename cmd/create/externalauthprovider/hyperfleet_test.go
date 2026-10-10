package externalauthprovider

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Platform API external authentication provider create", func() {
	var testRuntime *test.TestingRuntime

	BeforeEach(func() {
		testRuntime = test.NewTestRuntime()
		interactive.SetEnabled(false)
		DeferCleanup(func() { interactive.SetEnabled(false) })
		previousFlags := hfExternalAuthFlags
		hfExternalAuthFlags = struct {
			issuerCAName      string
			discoveryURL      string
			groupsClaimPrefix string
		}{
			issuerCAName:      "idp-ca",
			discoveryURL:      "https://discovery.example.com/.well-known/openid-configuration",
			groupsClaimPrefix: "corp:",
		}
		DeferCleanup(func() { hfExternalAuthFlags = previousFlags })
	})

	It("writes an OIDC provider to the ready HostedCluster", func() {
		cmd := &cobra.Command{Use: "external-auth-provider"}
		flags := cmd.Flags()
		flags.String("name", "corp", "")
		flags.String("issuer-url", "https://id.example.com", "")
		flags.StringSlice("issuer-audiences", []string{"openshift-api"}, "")
		flags.String("claim-mapping-username-claim", "email", "")
		flags.String("claim-mapping-groups-claim", "groups", "")
		flags.StringSlice("claim-validation-rule", []string{}, "")

		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		clusters := hfmocks.NewMockClusterInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().Clusters().Return(clusters).AnyTimes()
		clusters.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(&v1alpha1.ClusterList{
			Items: []v1alpha1.Cluster{{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}},
		}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", platform.GetOptions{}).Return(&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Status:     v1alpha1.ClusterStatus{Phase: v1alpha1.ClusterPhaseReady},
		}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster-uid", types.MergePatchType, gomock.Any(), platform.PatchOptions{}).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, patch []byte, _ platform.PatchOptions) (*v1alpha1.Cluster, error) {
				Expect(patch).To(MatchJSON(`{
					"spec":{"hostedCluster":{"configuration":{"authentication":{
						"type":"OIDC",
						"oidcProviders":[{
							"name":"corp",
							"issuer":{
								"issuerURL":"https://id.example.com",
								"audiences":["openshift-api"],
								"issuerCertificateAuthority":{"name":"idp-ca"},
								"discoveryURL":"https://discovery.example.com/.well-known/openid-configuration"
							},
							"claimMappings":{
								"username":{"claim":"email","prefixPolicy":"NoPrefix"},
								"groups":{"claim":"groups","prefix":"corp:"}
							}
						}]
					}}}}}`))
				return &v1alpha1.Cluster{}, nil
			})

		testRuntime.RosaRuntime.HyperFleetClient = hf
		ocm.SetClusterKey("cluster1")
		Expect(runHyperfleetExternalAuthProvider(context.Background(), testRuntime.RosaRuntime, cmd)).To(Succeed())
	})
})
