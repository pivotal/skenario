package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakes "k8s.io/client-go/kubernetes/fake"
)

const (
	stableWindow           = 60 * time.Second
	panicWindow            = 6 * time.Second
	scaleToZeroGracePeriod = 30 * time.Second
	testNamespace = "simulator-namespace"
)

var (
	informerFactory   informers.SharedInformerFactory
	endpointsInformer v1.EndpointsInformer
	fakeClient        kubernetes.Interface
)

func main() {
	fmt.Println("starting")

	unsugaredLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("config error!!1!: %s", err.Error())
	}
	logger := unsugaredLogger.Sugar()

	fakeClient = fakes.NewSimpleClientset()
	informerFactory = informers.NewSharedInformerFactory(fakeClient, 0)
	endpointsInformer = informerFactory.Core().V1().Endpoints()

	config := &autoscaler.Config{
		ContainerConcurrencyTargetPercentage: 1.0, // targeting 100% makes the test easier to read
		ContainerConcurrencyTargetDefault:    15.0,
		MaxScaleUpRate:                       100.0,
		StableWindow:                         stableWindow,
		PanicWindow:                          panicWindow,
		ScaleToZeroGracePeriod:               scaleToZeroGracePeriod,
	}

	dynConfig := autoscaler.NewDynamicConfig(config, logger)

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		"revisionService",
		endpointsInformer,
		100.0,
		&mockReporter{},
	)
	ctx := context.TODO()

	t := time.Now()

	w := csv.NewWriter(os.Stdout)

	err = w.Write([]string{"time", "avg_concurrent_requests", "desired_replicas"})
	if err != nil {
		logger.Fatal("could not write header: %s", err.Error())
	}

	var stepper SimStepper
	stepper = &linear{step: 0}

	for i := int32(0); i < 200; i++ {
		stepper.Step(int(i))

		t = t.Add(time.Second)
		avgConcurrent := float64(stepper.AverageConcurrent())
		//reqCount := i

		stat := autoscaler.Stat{
			Time:                      &t,
			PodName:                   fmt.Sprintf("simulator-pod-%d", i),
			AverageConcurrentRequests: avgConcurrent,
			//RequestCount:              reqCount,
			//LameDuck:                  false,
		}
		as.Record(ctx, stat)
		desired, _ := as.Scale(ctx, t)

		createEndpoints(addIps(makeEndpoints(), int(stepper.RunningPods())))

		err = w.Write([]string{
			strconv.Itoa(int(t.Unix())),
			strconv.FormatFloat(avgConcurrent, 'f', 2, 64),
			//strconv.Itoa(int(reqCount)),
			strconv.Itoa(int(desired)),
		})

		if err != nil {
			logger.Fatal("could not write record: %s", err.Error())
		}
	}
	w.Flush()
}

type SimStepper interface {
	AverageConcurrent() float64
	RunningPods() int
	Step(step int)
}

type linear struct {
	step int
}

func (l *linear) AverageConcurrent() float64 {
	return float64(l.step)
}

func (l *linear) RunningPods() int {
	return l.step
}

func (l *linear) Step(step int) {
	if step < 50 {
		l.step = step
	}
}

type mockReporter struct{}

// ReportDesiredPodCount of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportDesiredPodCount(v int64) error {
	return nil
}

// ReportRequestedPodCount of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportRequestedPodCount(v int64) error {
	return nil
}

// ReportActualPodCount of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportActualPodCount(v int64) error {
	return nil
}

// ReportObservedPodCount of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportObservedPodCount(v float64) error {
	return nil
}

// ReportStableRequestConcurrency of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportStableRequestConcurrency(v float64) error {
	return nil
}

// ReportPanicRequestConcurrency of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportPanicRequestConcurrency(v float64) error {
	return nil
}

// ReportTargetRequestConcurrency of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportTargetRequestConcurrency(v float64) error {
	return nil
}

// ReportPanic of a mockReporter does nothing and return nil for error.
func (r *mockReporter) ReportPanic(v int64) error {
	return nil
}

func makeEndpoints() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "revisionService",
		},
	}
}

func addIps(ep *corev1.Endpoints, ipCount int) *corev1.Endpoints {
	epAddresses := []corev1.EndpointAddress{}
	for i := 1; i <= ipCount; i++ {
		ip := fmt.Sprintf("127.0.0.%v", i)
		epAddresses = append(epAddresses, corev1.EndpointAddress{IP: ip})
	}
	ep.Subsets = []corev1.EndpointSubset{{
		Addresses: epAddresses,
	}}
	return ep
}

func createEndpoints(ep *corev1.Endpoints) {
	fakeClient.CoreV1().Endpoints(testNamespace).Create(ep)
	endpointsInformer.Informer().GetIndexer().Add(ep)
}
