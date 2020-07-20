package plugin

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	scalefake "k8s.io/client-go/scale/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	cmfake "k8s.io/metrics/pkg/client/custom_metrics/fake"
	emfake "k8s.io/metrics/pkg/client/external_metrics/fake"

	podautoscaler "k8s.io/kubernetes/pkg/controller/podautoscaler"

	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

type Autoscaler struct {
	mux        sync.RWMutex
	controller *podautoscaler.HorizontalController
	hpa        *autoscalingv1.HorizontalPodAutoscaler
	pods       map[string]*proto.Pod
	stats      map[string]*proto.Stat
}

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

func NewAutoscaler(hpaYaml string) (*Autoscaler, error) {

	client := &fake.Clientset{}
	evtNamespacer := client.CoreV1()
	scaleNamespacer := &scalefake.FakeScaleClient{}
	hpaNamespacer := client.AutoscalingV1()
	mapper := testrestmapper.TestOnlyStaticRESTMapper(legacyscheme.Scheme)
	testMetricsClient := &metricsfake.Clientset{}
	testCMClient := &cmfake.FakeCustomMetricsClient{}
	testEMClient := &emfake.FakeExternalMetricsClient{}
	metricsClient := metrics.NewRESTMetricsClient(
		testMetricsClient.MetricsV1beta1(),
		testCMClient,
		testEMClient,
	)
	informerFactory := informers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	hpaInformer := informerFactory.Autoscaling().V1().HorizontalPodAutoscalers()

	autoscaler := &Autoscaler{}

	podInformer := &fakePodInformer{
		lister: &fakePodLister{
			autoscaler: autoscaler,
		},
		informer: &fakeSharedIndexInformer{},
	}

	resyncPeriod := controller.NoResyncPeriodFunc()
	downscaleStabilizationWindow := 5 * time.Minute
	tolerance := 0.1
	cpuInitializationPeriod := 2 * time.Minute
	delayOfInitialReadinessStatus := 10 * time.Second

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(hpaYaml)), 1000)
	if err := decoder.Decode(&hpa); err != nil {
		return nil, err
	}
	hpaRaw, err := unsafeConvertToVersionVia(hpa, autoscalingv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	hpav1 := hpaRaw.(*autoscalingv1.HorizontalPodAutoscaler)

	autoscaler.controller = podautoscaler.NewHorizontalController(
		evtNamespacer,
		scaleNamespacer,
		hpaNamespacer,
		mapper,
		metricsClient,
		hpaInformer,
		podInformer,
		resyncPeriod,
		downscaleStabilizationWindow,
		tolerance,
		cpuInitializationPeriod,
		delayOfInitialReadinessStatus,
	)
	autoscaler.hpa = hpav1
	autoscaler.pods = make(map[string]*proto.Pod)
	autoscaler.stats = make(map[string]*proto.Stat)

	client.AddReactor("update", "horizontalpodautoscalers", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("update horizontalpodautoscaler")
		autoscaler.hpa = action.(core.UpdateAction).GetObject().(*autoscalingv1.HorizontalPodAutoscaler)
		return true, nil, nil
	})
	client.AddReactor("list", "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
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
	scaleNamespacer.AddReactor("get", "deployments", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("get deployments")
		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hpav1.Name,
				Namespace: hpav1.Namespace,
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
	testMetricsClient.AddReactor("list", "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		// log.Printf("metrics list pods\n")
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

	return autoscaler, nil
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

func (a *Autoscaler) Scale(now int64) (int32, error) {
	a.mux.Lock()
	defer a.mux.Unlock()
	if err := a.controller.ReconcileAutoscaler(time.Unix(0, now), a.hpa, "hpa"); err != nil {
		return 0, err
	}
	return a.hpa.Status.DesiredReplicas, nil
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
	return fmt.Sprintf("+%v", a.hpa)
}

// Forked from horizontal.go.
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
