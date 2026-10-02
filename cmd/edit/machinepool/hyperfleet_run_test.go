package machinepool

import (
	"context"
	"fmt"
	"math"

	"go.uber.org/mock/gomock"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

func newEditMPMocks(ctrl *gomock.Controller) (
	*hfmocks.MockInterface,
	*hfmocks.MockClusterInterface,
	*hfmocks.MockNodePoolInterface,
) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	v1.EXPECT().NodePools(gomock.Any()).Return(nodePools).AnyTimes()
	return hf, clusters, nodePools
}

func makeEditCmd(replicasStr string) *cobra.Command {
	cmd := NewEditMachinePoolCommand()
	if err := cmd.Flag("cluster").Value.Set("cluster1"); err != nil {
		panic(err)
	}
	if replicasStr != "" {
		if err := cmd.Flags().Set("replicas", replicasStr); err != nil {
			panic(err)
		}
	}
	return cmd
}

func makeEditNodePool(replicas *int32, autoscaling *hypershiftv1beta1.NodePoolAutoScaling) *v1alpha1.NodePool {
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
		Spec: v1alpha1.NodePoolSpec{
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				Replicas:    replicas,
				AutoScaling: autoscaling,
			},
		},
	}
}

func expectEditNodePoolResolution(clusters *hfmocks.MockClusterInterface, nodePools *hfmocks.MockNodePoolInterface,
	np *v1alpha1.NodePool) {
	clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
	}}}, nil)
	nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
}

var _ = Describe("runHyperfleetEdit (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("updates replicas on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
			makeEditCmd("5"), nil)
	})

	It("resolves node pool name from argv when machinepool option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{replicas: 5},
			makeEditCmd("5"), []string{"my-np"})
	})

	It("fails when machinepool name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{}, makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("fails when no supported flags are changed", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		// ResolveClusterUID and ResolveNodePoolUID are called before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
		}}}, nil)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
				makeEditCmd(""), nil)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		cmd := NewEditMachinePoolCommand()
		if err := cmd.Flags().Set("replicas", "5"); err != nil {
			panic(err)
		}
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"}, cmd, nil)
		}).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newEditMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np"},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	// setupListOnlyMocks sets up cluster/nodepool List mocks but NOT Get.
	// Use for tests that fail in PreRequest (before PostExpand calls Get).
	setupListOnlyMocks := func(ctrl *gomock.Controller) {
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		np := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		t.RosaRuntime.HyperFleetClient = hf
	}

	It("rejects negative replica count", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// PreRequest fails before PostExpand — Get is never called.
		setupListOnlyMocks(ctrl)

		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime,
				&EditMachinepoolUserOptions{machinepool: "my-np", replicas: -1},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("rejects replica count above math.MaxInt32", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// PreRequest fails before PostExpand — Get is never called.
		setupListOnlyMocks(ctrl)

		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime,
				&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32 + 1},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})

	It("accepts math.MaxInt32 as a valid replica count", func() {
		ctrl := gomock.NewController(GinkgoT())
		_, nodePools, np := func() (*hfmocks.MockInterface, *hfmocks.MockNodePoolInterface, *v1alpha1.NodePool) {
			hf, clusters, nps := newEditMPMocks(ctrl)
			replicas := int32(3)
			np := &v1alpha1.NodePool{
				ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
				Spec: v1alpha1.NodePoolSpec{
					NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
				},
			}
			clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}}}, nil)
			nps.EXPECT().List(gomock.Any(), gomock.Any()).Return(
				&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
			nps.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
			nps.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(np, nil)
			t.RosaRuntime.HyperFleetClient = hf
			return hf, nps, np
		}()
		_ = nodePools
		_ = np

		runHyperfleetEdit(t.RosaRuntime,
			&EditMachinepoolUserOptions{machinepool: "my-np", replicas: math.MaxInt32},
			makeEditCmd("5"), nil)
	})

	It("sets fixed replicas to explicit zero when autoscaling is disabled", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		minReplicas := int32(1)
		current := makeEditNodePool(nil, &hypershiftv1beta1.NodePoolAutoScaling{Min: &minReplicas, Max: 3})
		expectEditNodePoolResolution(clusters, nodePools, current)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)

		var updated *v1alpha1.NodePool
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, obj *v1alpha1.NodePool, _ platform.UpdateOptions) (*v1alpha1.NodePool, error) {
				updated = obj.DeepCopy()
				return obj, nil
			})

		cmd := makeEditCmd("0")
		Expect(cmd.Flags().Set("enable-autoscaling", "false")).To(Succeed())
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{
			machinepool: "my-np", replicas: 0, autoscalingEnabled: false,
		}, cmd, nil)

		Expect(updated).ToNot(BeNil())
		Expect(updated.Spec.NodePool.AutoScaling).To(BeNil())
		Expect(updated.Spec.NodePool.Replicas).ToNot(BeNil())
		Expect(*updated.Spec.NodePool.Replicas).To(BeZero())
	})

	It("requires explicit autoscaling disablement before setting replicas on an autoscaled pool", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newEditMPMocks(ctrl)
		minReplicas := int32(1)
		current := makeEditNodePool(nil, &hypershiftv1beta1.NodePoolAutoScaling{Min: &minReplicas, Max: 3})
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)
		t.RosaRuntime.HyperFleetClient = hf

		cmd := makeEditCmd("0")
		input := &hfpathbind.NodePoolUpdateInput{}
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "np-uid-1", nodePoolKey: "my-np",
			userOptions: &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 0}, cmd: cmd,
		}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		obj := &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Replicas: input.Replicas,
		}}}
		err := handler.PostExpand(context.Background(), t.RosaRuntime, input, obj)
		Expect(err).To(MatchError(ContainSubstring("disable it before setting replicas")))
	})

	It("rejects replicas when autoscaling is explicitly enabled", func() {
		cmd := makeEditCmd("0")
		Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
		handler := &hyperfleetNodePoolUpdate{
			userOptions: &EditMachinepoolUserOptions{replicas: 0, autoscalingEnabled: true}, cmd: cmd,
		}
		err := handler.PreRequest(context.Background(), t.RosaRuntime, &hfpathbind.NodePoolUpdateInput{})
		Expect(err).To(MatchError("replicas cannot be set when autoscaling is enabled"))
	})

	It("rejects a zero autoscaling maximum", func() {
		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("max-replicas", "0")).To(Succeed())
		handler := &hyperfleetNodePoolUpdate{
			userOptions: &EditMachinepoolUserOptions{maxReplicas: 0}, cmd: cmd,
		}
		err := handler.PreRequest(context.Background(), t.RosaRuntime, &hfpathbind.NodePoolUpdateInput{})
		Expect(err).To(MatchError("max-replicas must be greater than zero"))
	})

	It("enables autoscaling with a zero minimum and omits fixed replicas", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)
		fixedReplicas := int32(1)
		current := makeEditNodePool(&fixedReplicas, nil)
		expectEditNodePoolResolution(clusters, nodePools, current)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)

		var updated *v1alpha1.NodePool
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, obj *v1alpha1.NodePool, _ platform.UpdateOptions) (*v1alpha1.NodePool, error) {
				updated = obj.DeepCopy()
				return obj, nil
			})

		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
		Expect(cmd.Flags().Set("min-replicas", "0")).To(Succeed())
		Expect(cmd.Flags().Set("max-replicas", "2")).To(Succeed())
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{
			machinepool: "my-np", autoscalingEnabled: true, minReplicas: 0, maxReplicas: 2,
		}, cmd, nil)

		Expect(updated).ToNot(BeNil())
		Expect(updated.Spec.NodePool.Replicas).To(BeNil())
		Expect(updated.Spec.NodePool.AutoScaling).ToNot(BeNil())
		Expect(updated.Spec.NodePool.AutoScaling.Min).ToNot(BeNil())
		Expect(*updated.Spec.NodePool.AutoScaling.Min).To(BeZero())
		Expect(updated.Spec.NodePool.AutoScaling.Max).To(Equal(int32(2)))
	})

	DescribeTable("preserves the unchanged autoscaling bound", func(flag, value string, wantMin, wantMax int32) {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newEditMPMocks(ctrl)
		currentMin := int32(2)
		current := makeEditNodePool(nil, &hypershiftv1beta1.NodePoolAutoScaling{Min: &currentMin, Max: 5})
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)
		t.RosaRuntime.HyperFleetClient = hf

		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set(flag, value)).To(Succeed())
		options := &EditMachinepoolUserOptions{machinepool: "my-np"}
		if flag == "min-replicas" {
			options.minReplicas = 1
		} else {
			options.maxReplicas = 8
		}
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "np-uid-1", nodePoolKey: "my-np",
			userOptions: options, cmd: cmd,
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		obj := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime, input, obj)).To(Succeed())
		Expect(obj.Spec.NodePool.AutoScaling).ToNot(BeNil())
		Expect(*obj.Spec.NodePool.AutoScaling.Min).To(Equal(wantMin))
		Expect(obj.Spec.NodePool.AutoScaling.Max).To(Equal(wantMax))
		Expect(obj.Spec.NodePool.Replicas).To(BeNil())
	},
		Entry("min-only edit", "min-replicas", "1", int32(1), int32(5)),
		Entry("max-only edit", "max-replicas", "8", int32(2), int32(8)),
	)

	It("rejects autoscaling bounds on a fixed pool", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newEditMPMocks(ctrl)
		fixedReplicas := int32(1)
		current := makeEditNodePool(&fixedReplicas, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)
		t.RosaRuntime.HyperFleetClient = hf

		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("min-replicas", "0")).To(Succeed())
		Expect(cmd.Flags().Set("max-replicas", "2")).To(Succeed())
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "np-uid-1", nodePoolKey: "my-np",
			userOptions: &EditMachinepoolUserOptions{machinepool: "my-np", minReplicas: 0, maxReplicas: 2}, cmd: cmd,
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime, input, &v1alpha1.NodePool{})).To(
			MatchError("autoscaling must be enabled in order to set min and max replicas"))
	})

	It("validates the final range when editing only one autoscaling bound", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _, nodePools := newEditMPMocks(ctrl)
		currentMin := int32(2)
		current := makeEditNodePool(nil, &hypershiftv1beta1.NodePoolAutoScaling{Min: &currentMin, Max: 5})
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(current, nil)
		t.RosaRuntime.HyperFleetClient = hf

		cmd := makeEditCmd("")
		Expect(cmd.Flags().Set("min-replicas", "6")).To(Succeed())
		handler := &hyperfleetNodePoolUpdate{
			clusterUID: "cluster-uid", nodePoolUID: "np-uid-1", nodePoolKey: "my-np",
			userOptions: &EditMachinepoolUserOptions{machinepool: "my-np", minReplicas: 6}, cmd: cmd,
		}
		input := &hfpathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), t.RosaRuntime, input)).To(Succeed())
		Expect(handler.PostExpand(context.Background(), t.RosaRuntime, input, &v1alpha1.NodePool{})).To(
			MatchError("max-replicas must be greater than or equal to min-replicas"))
	})

	It("fails when node pool update fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newEditMPMocks(ctrl)

		replicas := int32(3)
		np := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: types.UID("np-uid-1")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas},
			},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{*np}}, nil)
		nodePools.EXPECT().Get(gomock.Any(), "np-uid-1", gomock.Any()).Return(np, nil)
		nodePools.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetEdit(t.RosaRuntime, &EditMachinepoolUserOptions{machinepool: "my-np", replicas: 5},
				makeEditCmd("5"), nil)
		}).To(Panic())
	})
})
