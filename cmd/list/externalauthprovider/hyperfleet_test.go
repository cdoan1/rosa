package externalauthprovider

import (
	"context"
	"encoding/json"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	configv1 "github.com/openshift/api/config/v1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Platform API external authentication provider list", func() {
	var testRuntime *test.TestingRuntime

	BeforeEach(func() {
		testRuntime = test.NewTestRuntime()
		output.SetOutput("")
		DeferCleanup(output.SetOutput, "")
	})

	It("lists the HostedCluster OIDC provider", func() {
		provider := testOIDCProvider()
		stubExternalAuthCluster(gomock.NewController(GinkgoT()), testRuntime, provider)

		stdout := captureHyperfleetListStdout(func() {
			Expect(runHyperfleetList(context.Background(), testRuntime.RosaRuntime, Cmd)).To(Succeed())
		})
		Expect(stdout).To(ContainSubstring("NAME"))
		Expect(stdout).To(ContainSubstring("corp"))
		Expect(stdout).To(ContainSubstring("https://id.example.com"))
	})

	It("prints the configured provider as JSON", func() {
		provider := testOIDCProvider()
		stubExternalAuthCluster(gomock.NewController(GinkgoT()), testRuntime, provider)
		output.SetOutput(output.JSON)

		stdout := captureHyperfleetListStdout(func() {
			Expect(runHyperfleetList(context.Background(), testRuntime.RosaRuntime, Cmd)).To(Succeed())
		})
		var providers []configv1.OIDCProvider
		Expect(json.Unmarshal([]byte(stdout), &providers)).To(Succeed())
		Expect(providers).To(HaveLen(1))
		Expect(providers[0].Name).To(Equal("corp"))
		Expect(providers[0].Issuer.URL).To(Equal("https://id.example.com"))
	})
})

func testOIDCProvider() configv1.OIDCProvider {
	return configv1.OIDCProvider{
		Name: "corp",
		Issuer: configv1.TokenIssuer{
			URL:       "https://id.example.com",
			Audiences: []configv1.TokenAudience{"openshift-api"},
		},
		ClaimMappings: configv1.TokenClaimMappings{
			Username: configv1.UsernameClaimMapping{Claim: "email", PrefixPolicy: configv1.NoPrefix},
			Groups: configv1.PrefixedClaimMapping{
				TokenClaimMapping: configv1.TokenClaimMapping{Claim: "groups"},
				Prefix:            "corp:",
			},
		},
	}
}

func stubExternalAuthCluster(ctrl *gomock.Controller, t *test.TestingRuntime, provider configv1.OIDCProvider) {
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
		Spec: v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
			Configuration: &v1alpha1.ClusterConfiguration{Authentication: &v1alpha1.ClusterAuthentication{
				Type: "OIDC", OIDCProviders: []configv1.OIDCProvider{provider},
			}},
		}},
		Status: v1alpha1.ClusterStatus{Phase: v1alpha1.ClusterPhaseReady},
	}, nil)
	t.RosaRuntime.HyperFleetClient = hf
}

func captureHyperfleetListStdout(f func()) string {
	r, w, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())
	original := os.Stdout
	os.Stdout = w
	f()
	Expect(w.Close()).To(Succeed())
	os.Stdout = original
	data, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
