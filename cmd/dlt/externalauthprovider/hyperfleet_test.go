package externalauthprovider

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	configv1 "github.com/openshift/api/config/v1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Platform API external authentication provider delete", func() {
	var testRuntime *test.TestingRuntime

	BeforeEach(func() {
		testRuntime = test.NewTestRuntime()
		originalConfirm := hfConfirmExternalAuthProvider
		hfConfirmExternalAuthProvider = func(string, ...interface{}) bool { return true }
		DeferCleanup(func() { hfConfirmExternalAuthProvider = originalConfirm })
	})

	It("removes the named provider and resets the authentication type", func() {
		ctrl := gomock.NewController(GinkgoT())
		clusters := stubDeleteExternalAuthCluster(ctrl, testRuntime)
		clusters.EXPECT().Patch(gomock.Any(), "cluster-uid", types.MergePatchType, gomock.Any(), platform.PatchOptions{}).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ platform.PatchOptions) (*v1alpha1.Cluster, error) {
				var patch map[string]any
				Expect(json.Unmarshal(data, &patch)).To(Succeed())
				authentication := patch["spec"].(map[string]any)["hostedCluster"].(map[string]any)["configuration"].(map[string]any)["authentication"].(map[string]any)
				Expect(authentication).To(HaveKeyWithValue("type", BeNil()))
				Expect(authentication).To(HaveKeyWithValue("oidcProviders", BeNil()))
				return &v1alpha1.Cluster{}, nil
			})

		ocm.SetClusterKey("cluster1")
		Expect(runHyperfleetDelete(context.Background(), testRuntime.RosaRuntime, []string{"corp"})).To(Succeed())
	})

	It("does not patch when the user declines the confirmation", func() {
		stubDeleteExternalAuthCluster(gomock.NewController(GinkgoT()), testRuntime)
		hfConfirmExternalAuthProvider = func(string, ...interface{}) bool { return false }
		ocm.SetClusterKey("cluster1")

		Expect(runHyperfleetDelete(context.Background(), testRuntime.RosaRuntime, []string{"corp"})).To(Succeed())
	})
})

func stubDeleteExternalAuthCluster(ctrl *gomock.Controller, t *test.TestingRuntime) *hfmocks.MockClusterInterface {
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
				Type: "OIDC",
				OIDCProviders: []configv1.OIDCProvider{{
					Name:   "corp",
					Issuer: configv1.TokenIssuer{URL: "https://id.example.com"},
				}},
			}},
		}},
		Status: v1alpha1.ClusterStatus{Phase: v1alpha1.ClusterPhaseReady},
	}, nil)
	t.RosaRuntime.HyperFleetClient = hf
	return clusters
}
