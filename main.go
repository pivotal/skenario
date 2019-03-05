package main

import (
	"context"
	"fmt"
	"github.com/wcharczuk/go-chart"
	"os"
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
	testNamespace          = "simulator-namespace"
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
		// ContainerConcurrencyTargetDefault:    50.0,
		MaxScaleUpRate:         10.0,
		StableWindow:           stableWindow,
		PanicWindow:            panicWindow,
		ScaleToZeroGracePeriod: scaleToZeroGracePeriod,
	}

	dynConfig := autoscaler.NewDynamicConfig(config, logger)

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		"revisionService",
		endpointsInformer,
		10.0,
		&mockReporter{},
	)
	ctx := context.TODO()

	t := time.Now()

	// w := csv.NewWriter(os.Stdout)
	// err = w.Write([]string{"time", "avg_concurrent_requests", "desired_replicas"})
	// if err != nil {
	// 	logger.Fatal("could not write header: %s", err.Error())
	// }

	var stepper SimStepper
	stepper = &linear{step: 0}
	steps := int32(1000)

	ch := chart.Chart{
		Title: fmt.Sprintf("Autoscaler Simulation %d", time.Now().UTC().Unix()),
		TitleStyle: chart.Style{
			Show: true,
		},
		Background: chart.Style{
			Show: false,
			Padding: chart.Box{
				Top:    80,
				Left:   40,
				Right:  20,
				Bottom: 20,
				IsSet:  false,
			},
		},
		XAxis: chart.XAxis{
			Name:      "Time",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Range: &chart.ContinuousRange{
				Min:        0,
				Max:        float64(steps),
				Domain:     0,
				Descending: false,
			},
		},
		YAxis: chart.YAxis{
			Name:      "Avg Concurrent",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Range: &chart.ContinuousRange{
				Min:        0,
				Max:        30,
				Domain:     0,
				Descending: false,
			},
		},
		YAxisSecondary: chart.YAxis{
			Name:      "Desired Replicas",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Range: &chart.ContinuousRange{
				Min:        0,
				Max:        16,
				Domain:     0,
				Descending: false,
			},
		},
	}

	ch.Elements = []chart.Renderable{
		chart.LegendLeft(&ch),
	}

	desiredPoints := chart.ContinuousSeries{
		Name:  "Desired Replicas",
		Style: chart.StyleShow(),
	}
	runningPoints := chart.ContinuousSeries{
		Name:  "Running Replicas",
		Style: chart.StyleShow(),
	}
	concurrentPoints := chart.ContinuousSeries{
		Name:  "Avg Concurrency",
		Style: chart.StyleShow(),
	}

	for i := int32(0); i < steps; i++ {
		stepper.Step(int(i))

		t = t.Add(time.Second)
		avgConcurrent := stepper.AverageConcurrent()
		reqCount := stepper.RequestCount()

		for j := 0; j < stepper.RunningPods(); j++ {
			stat := autoscaler.Stat{
				Time:                      &t,
				PodName:                   fmt.Sprintf("simulator-pod-%d", j),
				AverageConcurrentRequests: avgConcurrent,
				RequestCount:              int32(reqCount),
			}
			as.Record(ctx, stat)
		}
		desired, _ := as.Scale(ctx, t)

		createEndpoints(addIps(makeEndpoints(), int(stepper.RunningPods())))

		// err = w.Write([]string{
		// 	strconv.Itoa(int(t.Unix())),
		// 	strconv.FormatFloat(avgConcurrent, 'f', 2, 64),
		// 	// strconv.Itoa(int(reqCount)),
		// 	strconv.Itoa(int(desired)),
		// })
		// if err != nil {
		// 	logger.Fatalf("could not write record: %s", err.Error())
		// }

		desiredPoints.XValues = append(desiredPoints.XValues, float64(i))
		desiredPoints.YValues = append(desiredPoints.YValues, float64(desired))

		runningPoints.XValues = append(runningPoints.XValues, float64(i))
		runningPoints.YValues = append(runningPoints.YValues, float64(stepper.RunningPods()))

		concurrentPoints.XValues = append(concurrentPoints.XValues, float64(i))
		concurrentPoints.YValues = append(concurrentPoints.YValues, float64(avgConcurrent))
	}
	// w.Flush()

	ch.Series = []chart.Series{desiredPoints, runningPoints, concurrentPoints}

	pngFile, err := os.OpenFile("chart.png", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		logger.Fatalf("could not open or create chart file: %s", err.Error())
	}

	err = ch.Render(chart.PNG, pngFile)
	if err != nil {
		logger.Fatalf("could not render chart: %s", err.Error())
	}
}

type SimStepper interface {
	AverageConcurrent() float64
	RequestCount() int
	RunningPods() int
	Step(step int)
}

type linear struct {
	step        int
	lastDesired int32
}

func (l *linear) AverageConcurrent() float64 {
	return float64(l.RequestCount()) / float64(l.RunningPods())
}

func (l *linear) RequestCount() int {
	if l.step < 100 {
		return 10
	} else if l.step < 200 {
		return 20
	} else if l.step < 300 {
		return 40
	} else if l.step < 600 {
		return 80
	}

	return 10
}

func (l *linear) RunningPods() int {
	if l.step < 100 {
		return 1
	} else if l.step < 250 {
		return 2
	} else if l.step < 450 {
		return 3
	} else if l.step < 650 {
		return 4
	}

	return 4
}

func (l *linear) Step(step int) {
	l.step = step
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
