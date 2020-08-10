package plugin

import (
	"bytes"
	"flag"
	"fmt"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/fake"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/checkpoint"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/input"
	controllerfetcher "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/input/controller_fetcher"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/input/metrics"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/input/oom"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/logic"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/model"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/target"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"

	//coreinformers "k8s.io/client-go/informers/core/v1"
	//corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	scalefake "k8s.io/client-go/scale/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"strconv"

	//"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	//metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	//cmfake "k8s.io/metrics/pkg/client/custom_metrics/fake"
	//emfake "k8s.io/metrics/pkg/client/external_metrics/fake"
	"log"

	//"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/utils/metrics/recommender"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"sync"
	"time"

	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"

	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"

	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpa_clientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	lister "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/listers/autoscaling.k8s.io/v1"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/routines"
	kube_client "k8s.io/client-go/kubernetes"
	fakemetricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1/fake"
)

type Autoscaler struct {
	mux         sync.RWMutex
	recommender routines.Recommender
	vpa         *vpav1.VerticalPodAutoscaler
	pods        map[string]*proto.Pod
	stats       map[string]*proto.Stat
}

var checkpointsGCInterval = flag.Duration("checkpoints-gc-interval", 10*time.Minute, `How often orphaned checkpoints should be garbage collected`)

//var KubeApiQps = flag.Float64("kube-api-qps", 5.0, `QPS limit when making requests to Kubernetes apiserver`)
//var KubeApiBurst = flag.Float64("kube-api-burst", 10.0, `QPS burst limit when making requests to Kubernetes apiserver`)

// Create a non-concurrent, non-cached informer for simulation.

var _ coreinformers.PodInformer = &fakePodInformer{}

type fakePodInformer struct {
	lister   corelisters.PodLister
	informer cache.SharedIndexInformer
}

func (f *fakePodInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *fakePodInformer) Lister() corelisters.PodLister {
	return f.lister
}

type fakeSharedIndexInformer struct{}

func (f *fakeSharedIndexInformer) AddEventHandler(_ cache.ResourceEventHandler) {}
func (f *fakeSharedIndexInformer) AddEventHandlerWithResyncPeriod(_ cache.ResourceEventHandler, _ time.Duration) {
}
func (f *fakeSharedIndexInformer) GetStore() cache.Store {
	panic("unimplemented")
}
func (f *fakeSharedIndexInformer) GetController() cache.Controller {
	panic("unimplemented")
}
func (f *fakeSharedIndexInformer) Run(_ <-chan struct{}) {}
func (f *fakeSharedIndexInformer) HasSynced() bool {
	return true
}
func (f *fakeSharedIndexInformer) LastSyncResourceVersion() string {
	panic("unimplemented")
}
func (f *fakeSharedIndexInformer) AddIndexers(_ cache.Indexers) error {
	panic("unimplemented")
}
func (f *fakeSharedIndexInformer) GetIndexer() cache.Indexer {
	panic("unimplemented")
}

// This plugin doesn't support namespaces.
var _ corelisters.PodLister = &fakePodLister{}
var _ corelisters.PodNamespaceLister = &fakePodLister{}

type fakePodLister struct {
	autoscaler *Autoscaler
}

func (f *fakePodLister) List(selector labels.Selector) (ret []*v1.Pod, err error) {
	return f.autoscaler.listPods()
}

func (f *fakePodLister) Pods(namespace string) corelisters.PodNamespaceLister {
	return f
}

func (f *fakePodLister) Get(name string) (*v1.Pod, error) {
	panic("unimplemented")
}

type fakePodObserver struct {
	autoscaler *Autoscaler
}

type fakeConfig struct {
}

func (c *fakeConfig) GoString() string {
	panic("unimplemented")
}

func (c *fakeConfig) String() string {
	panic("unimplemented")
}

type fakeVerticalPodAutoscalerLister struct {
	autoscaler *Autoscaler
}

// List lists all VerticalPodAutoscalers in the indexer.
func (s *fakeVerticalPodAutoscalerLister) List(selector labels.Selector) (ret []*vpav1.VerticalPodAutoscaler, err error) {
	ret = make([]*vpav1.VerticalPodAutoscaler, 0)
	ret = append(ret, s.autoscaler.vpa)
	return ret, nil
}

// VerticalPodAutoscalers returns an object that can list and get VerticalPodAutoscalers.
func (s *fakeVerticalPodAutoscalerLister) VerticalPodAutoscalers(namespace string) lister.VerticalPodAutoscalerNamespaceLister {
	return fakeVerticalPodAutoscalerNamespaceLister{s.autoscaler}
}

type fakeVerticalPodAutoscalerNamespaceLister struct {
	autoscaler *Autoscaler
}

func (s fakeVerticalPodAutoscalerNamespaceLister) List(selector labels.Selector) (ret []*vpav1.VerticalPodAutoscaler, err error) {
	ret = make([]*vpav1.VerticalPodAutoscaler, 0)
	ret = append(ret, s.autoscaler.vpa)
	return ret, nil
}

// Get retrieves the VerticalPodAutoscaler from the indexer for a given namespace and name.
func (s fakeVerticalPodAutoscalerNamespaceLister) Get(name string) (*vpav1.VerticalPodAutoscaler, error) {
	return s.autoscaler.vpa, nil
}

func NewAutoscaler(vpaYaml string) (*Autoscaler, error) {
	autoscaler := &Autoscaler{}
	vpa := &vpav1.VerticalPodAutoscaler{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(vpaYaml)), 1000)
	if err := decoder.Decode(&vpa); err != nil {
		return nil, err
	}
	//vpaRaw, err := unsafeConvertToVersionVia(vpa, vpav1.SchemeGroupVersion)
	//if err != nil {
	//	return nil, err
	//}
	//vpav11 := vpaRaw.(*vpav1.VerticalPodAutoscaler)

	autoscaler.vpa = vpa
	autoscaler.pods = make(map[string]*proto.Pod)
	autoscaler.stats = make(map[string]*proto.Stat)

	//config := CreateKubeConfig(float32(*KubeApiQps), int(*KubeApiBurst))
	config := &rest.Config{}
	//observedVpa := test.VerticalPodAutoscaler().WithName("vpa").WithNamespace("test").WithContainer("container").Get()
	//fakeMetricsGetter := fake.NewSimpleClientset(&vpa_types.VerticalPodAutoscalerList{Items: []vpa_types.VerticalPodAutoscaler{*observedVpa}})
	fakeMetricsGetter := &fake.Clientset{}
	fakeMetricsGetter.AddReactor("list", "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		log.Printf("metrics list pods\n")
		metrics := &metricsapi.PodMetricsList{}
		for _, pod := range autoscaler.pods {
			stat, ok := autoscaler.stats[pod.Name]
			var cpu int32 = 0
			if ok {
				cpu = stat.Value
			}
			podMetric := metricsapi.PodMetrics{
				ObjectMeta: metav1.ObjectMeta{
					Name:      stat.PodName,
					Namespace: "",
					Labels:    map[string]string{"key": "value"},
				},
				// TODO: get this (somehow) from Scale(now).
				Timestamp: metav1.Time{Time: time.Now()},
				Window:    metav1.Duration{Duration: time.Minute},
				Containers: []metricsapi.ContainerMetrics{
					{
						Name: "container",
						Usage: v1.ResourceList{
							v1.ResourceCPU: *resource.NewMilliQuantity(
								int64(cpu),
								resource.DecimalSI),
							v1.ResourceMemory: *resource.NewQuantity(
								int64(1024*1024),
								resource.BinarySI),
						},
					},
				},
			}
			metrics.Items = append(metrics.Items, podMetric)
		}

		return true, metrics, nil
	})
	fakeMetricsGetter.AddReactor("update", "verticalpodautoscalers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("update horizontalpodautoscaler")
		autoscaler.vpa = action.(core.UpdateAction).GetObject().(*vpav1.VerticalPodAutoscaler)
		return true, nil, nil
	})
	fakeMetricsGetter.AddReactor("list", "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		log.Printf("list pods")
		pods, err := autoscaler.listPods()
		if err != nil {
			return false, nil, err
		}
		obj := &v1.PodList{}
		for _, pod := range pods {
			obj.Items = append(obj.Items, *pod)
		}
		return true, obj, nil
	})

	//metricsClient := metrics.NewMetricsClient(fakeMetricsGetter.MetricsV1beta1())
	metricsClient := metrics.NewMetricsClient(&fakemetricsv1beta1.FakeMetricsV1beta1{Fake: &fakeMetricsGetter.Fake})
	clusterState := model.NewClusterState()

	autoscaler.recommender = routines.RecommenderFactory{
		ClusterState:       clusterState,
		ClusterStateFeeder: NewClusterStateFeeder(config, clusterState, false, metricsClient, autoscaler, vpa),
		//ClusterStateFeeder:     input.NewClusterStateFeederPlugin(config, clusterState, false, metricsClient),
		CheckpointWriter:       checkpoint.NewCheckpointWriter(clusterState, vpa_clientset.NewForConfigOrDie(config).AutoscalingV1()),
		VpaClient:              vpa_clientset.NewForConfigOrDie(config).AutoscalingV1(),
		PodResourceRecommender: logic.CreatePodResourceRecommender(),
		CheckpointsGCInterval:  *checkpointsGCInterval,
		UseCheckpoints:         true,
	}.Make()

	//scaleNamespacer.AddReactor("get", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
	//	// log.Printf("get deployments")
	//	obj := &autoscalingv1.Scale{
	//		ObjectMeta: metav1.ObjectMeta{
	//			Name:      hpav1.Name,
	//			Namespace: hpav1.Namespace,
	//		},
	//		Spec: autoscalingv1.ScaleSpec{
	//			Replicas: int32(len(autoscaler.pods)),
	//		},
	//		Status: autoscalingv1.ScaleStatus{
	//			// TODO: count of only ready pods.
	//			Replicas: int32(len(autoscaler.pods)),
	//			Selector: "key=value",
	//		},
	//	}
	//	return true, obj, nil
	//})
	//scaleNamespacer.AddReactor("update", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
	//	// log.Printf("update deployments scale")
	//	return false, nil, nil
	//})

	return autoscaler, nil
}

func (a *Autoscaler) Stat(stat []*proto.Stat) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	for _, s := range stat {
		//skip all metrics apart from cpu_millis
		if s.Type == proto.MetricType_CPU_MILLIS {
			a.stats[s.PodName] = s
		}
		// TODO: garbage collect stats after downscale stabilization window.
	}
	return nil
}

func (a *Autoscaler) VerticalRecommendation(now int64) ([]*proto.RecommendedPodResources, error) {
	a.mux.Lock()
	defer a.mux.Unlock()
	a.recommender.RunOnce(time.Unix(0, now))
	recommendation := make([]*proto.RecommendedPodResources, 0)
	if a.vpa.Status.Recommendation == nil {
		return recommendation, nil
	}
	for _, rec := range a.vpa.Status.Recommendation.ContainerRecommendations {

		//TODO make this part generic
		recommendation = append(recommendation, &proto.RecommendedPodResources{
			PodName:      rec.ContainerName,
			LowerBound:   rec.LowerBound.Cpu().Value(),
			UpperBound:   rec.UpperBound.Cpu().Value(),
			Target:       rec.Target.Cpu().Value(),
			ResourceName: v1.ResourceCPU.String(),
		})

		recommendation = append(recommendation, &proto.RecommendedPodResources{
			PodName:      rec.ContainerName,
			LowerBound:   rec.LowerBound.Memory().Value(),
			UpperBound:   rec.UpperBound.Memory().Value(),
			Target:       rec.Target.Memory().Value(),
			ResourceName: v1.ResourceMemory.String(),
		})
	}
	return recommendation, nil
}

func (a *Autoscaler) CreatePod(pod *proto.Pod) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if _, ok := a.pods[pod.Name]; ok {
		return fmt.Errorf("duplicate create pod event")
	}
	a.pods[pod.Name] = pod
	return nil
}

func (a *Autoscaler) UpdatePod(pod *proto.Pod) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if _, ok := a.pods[pod.Name]; !ok {
		return fmt.Errorf("update pod event for non-existant pod")
	}
	a.pods[pod.Name] = pod
	return nil
}

func (a *Autoscaler) DeletePod(pod *proto.Pod) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if _, ok := a.pods[pod.Name]; !ok {
		return fmt.Errorf("delete pod event for non-existant pod")
	}
	delete(a.pods, pod.Name)
	return nil
}

func (a *Autoscaler) String() string {
	return fmt.Sprintf("+%v", a.vpa)
}

//TODO not clear yet do we need it or not
func unsafeConvertToVersionVia(obj runtime.Object, externalVersion schema.GroupVersion) (runtime.Object, error) {
	objInt, err := legacyscheme.Scheme.UnsafeConvertToVersion(obj, schema.GroupVersion{Group: externalVersion.Group, Version: runtime.APIVersionInternal})
	if err != nil {
		return nil, fmt.Errorf("failed to convert the given object to the internal version: %v", err)
	}

	objExt, err := legacyscheme.Scheme.UnsafeConvertToVersion(objInt, externalVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the given object back to the external version: %v", err)
	}

	return objExt, err
}

func CreateKubeConfig(kubeApiQps float32, kubeApiBurst int) *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}
	config.QPS = kubeApiQps
	config.Burst = kubeApiBurst
	return config
}

func (a *Autoscaler) listPods() ([]*v1.Pod, error) {
	pods := make([]*v1.Pod, 0)
	// TODO: change phase based on proto state enum.
	podPhase := v1.PodRunning
	podReadiness := v1.ConditionTrue
	for _, pod := range a.pods {
		// TODO: pass pod start time in proto.
		podStartTime := metav1.NewTime(time.Unix(0, pod.LastTransition))
		pod := &v1.Pod{
			Status: v1.PodStatus{
				Phase: podPhase,
				Conditions: []v1.PodCondition{
					{
						Type:               v1.PodReady,
						Status:             podReadiness,
						LastTransitionTime: podStartTime,
					},
				},
				StartTime: &podStartTime,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: "",
				Labels: map[string]string{
					"key": "value",
				},
			},

			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU: resource.MustParse(strconv.Itoa(int(pod.CpuRequest)) + "m"),
							},
						},
					},
				},
			},
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func NewClusterStateFeeder(config *rest.Config, clusterState *model.ClusterState, memorySave bool, metricsClient metrics.MetricsClient, autoscaler *Autoscaler, vpav11 *vpav1.VerticalPodAutoscaler) input.ClusterStateFeeder {
	kubeClient := kube_client.NewForConfigOrDie(config)

	podLister, oomObserver := &fakePodLister{
		autoscaler: autoscaler,
	}, oom.NewObserver()
	//factory := informers.NewSharedInformerFactory(kubeClient, defaultResyncPeriod)
	//factory := informers.NewSharedInformerFactory(kubeClient, controller.NoResyncPeriodFunc())
	//controllerFetcher := controllerfetcher.NewControllerFetcher(config, kubeClient, factory)
	//controllerFetcher := &controllerfetcher.controllerFetcher{
	//	//scaleNamespacer: &scalefake.FakeScaleClient{},
	//	//mapper:          testrestmapper.TestOnlyStaticRESTMapper(legacyscheme.Scheme),
	//	//informersMap:    &fakeSharedIndexInformer{},
	//}
	scaleNamespacer := &scalefake.FakeScaleClient{}
	scaleNamespacer.AddReactor("get", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("get deployments")
		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vpav11.Name,
				Namespace: vpav11.Namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: int32(len(autoscaler.pods)),
			},
			Status: autoscalingv1.ScaleStatus{
				// TODO: count of only ready pods.
				Replicas: int32(len(autoscaler.pods)),
				Selector: "key=value",
			},
		}
		return true, obj, nil
	})
	scaleNamespacer.AddReactor("update", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("update deployments scale")
		return false, nil, nil
	})
	mapper := testrestmapper.TestOnlyStaticRESTMapper(legacyscheme.Scheme)
	informersMap := make(map[controllerfetcher.WellKnownController]cache.SharedIndexInformer)
	controllerFetcher := controllerfetcher.NewSimpleControllerFetcher(scaleNamespacer, mapper, informersMap)
	return input.ClusterStateFeederFactory{
		PodLister:           podLister,
		OOMObserver:         oomObserver,
		KubeClient:          kubeClient,
		MetricsClient:       metricsClient,
		VpaCheckpointClient: vpa_clientset.NewForConfigOrDie(config).AutoscalingV1(),
		VpaLister:           &fakeVerticalPodAutoscalerLister{autoscaler},
		ClusterState:        clusterState,
		SelectorFetcher:     target.NewSimpleVpaTargetSelectorFetcher(scaleNamespacer, mapper, make(map[target.WellKnownController]cache.SharedIndexInformer)),
		//SelectorFetcher:   &fakeVpaTargetSelectorFetcher{},
		MemorySaveMode:    memorySave,
		ControllerFetcher: controllerFetcher,
	}.Make()
}

type fakeVpaTargetSelectorFetcher struct {
}

func (f *fakeVpaTargetSelectorFetcher) Fetch(vpa *vpav1.VerticalPodAutoscaler) (labels.Selector, error) {
	return nil, nil
}

//func NewSimpleClientset(objects ...runtime.Object) *fake.Clientset {
//	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
//	for _, obj := range objects {
//		if err := o.Add(obj); err != nil {
//			panic(err)
//		}
//	}
//
//	cs := &fake.Clientset{tracker: o}
//	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
//	cs.AddReactor("*", "*", testing.ObjectReaction(o))
//	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
//		gvr := action.GetResource()
//		ns := action.GetNamespace()
//		watch, err := o.Watch(gvr, ns)
//		if err != nil {
//			return false, nil, err
//		}
//		return true, watch, nil
//	})
//
//	return cs
//}
