package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/prometheus/common/log"
	"github.com/wcharczuk/go-chart"
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
	targetConcurrency      = 5.0
	maxScaleUpRate         = 10.0
	testNamespace          = "simulator-namespace"
	testName               = "revisionService"
	steps                  = int32(1000)
)

var (
	informerFactory   informers.SharedInformerFactory
	endpointsInformer v1.EndpointsInformer
	fakeClient        kubernetes.Interface
	logger            *zap.SugaredLogger
	t                 time.Time
)

func main() {
	unsugaredLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("config error!!1!: %s", err.Error())
	}
	logger = unsugaredLogger.Sugar()

	fakeClient = fakes.NewSimpleClientset()
	informerFactory = informers.NewSharedInformerFactory(fakeClient, 0)
	endpointsInformer = informerFactory.Core().V1().Endpoints()

	config := &autoscaler.Config{
		MaxScaleUpRate:         maxScaleUpRate,
		StableWindow:           stableWindow,
		PanicWindow:            panicWindow,
		ScaleToZeroGracePeriod: scaleToZeroGracePeriod,
	}

	dynConfig := autoscaler.NewDynamicConfig(config, logger)

	statsReporter, err := autoscaler.NewStatsReporter(testNamespace, testName, "config-1", "revision-1")
	if err != nil {
		logger.Fatalf("could not create stats reporter: %s", err.Error())
	}

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		testName,
		endpointsInformer,
		targetConcurrency,
		statsReporter,
	)
	ctx := context.TODO()

	var stepper SimStepper
	stepper = &linear{step: 0}

	ch, desiredPoints, runningPoints, concurrentPoints := prepareChart(steps)

	createEndpoints(makeEndpoints())

	t = time.Unix(0, 0)
	for i := int32(0); i < steps; i++ {
		stepper.Step(int(i))

		t = t.Add(time.Second)
		for j := 0; j < stepper.RunningPods(); j++ {
			stat := autoscaler.Stat{
				Time:                      &t,
				PodName:                   fmt.Sprintf("simulator-pod-%d", j),
				AverageConcurrentRequests: stepper.AverageConcurrent(),
				RequestCount:              int32(stepper.RequestCount()),
			}
			as.Record(ctx, stat)
		}
		desired, _ := as.Scale(ctx, t)

		updateEndpoints(addIps(makeEndpoints(), int(stepper.RunningPods())))

		desiredPoints.XValues = append(desiredPoints.XValues, float64(i))
		desiredPoints.YValues = append(desiredPoints.YValues, float64(desired))

		runningPoints.XValues = append(runningPoints.XValues, float64(i))
		runningPoints.YValues = append(runningPoints.YValues, float64(stepper.RunningPods()))

		concurrentPoints.XValues = append(concurrentPoints.XValues, float64(i))
		concurrentPoints.YValues = append(concurrentPoints.YValues, float64(stepper.AverageConcurrent()))
	}

	ch.Series = []chart.Series{desiredPoints, runningPoints, concurrentPoints}

	pngFile, err := os.OpenFile("chart.png", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		logger.Fatalf("could not open or create chart.png file: %s", err.Error())
	}

	err = ch.Render(chart.PNG, pngFile)
	if err != nil {
		logger.Fatalf("could not render chart: %s", err.Error())
	}
}

func prepareChart(steps int32) (chart.Chart, chart.ContinuousSeries, chart.ContinuousSeries, chart.ContinuousSeries) {
	ch := chart.Chart{
		Title: fmt.Sprintf("Autoscaler Simulation %d", time.Now().UTC().Unix()),
		TitleStyle: chart.Style{
			Show: true,
		},
		Width:  1200,
		Height: 800,
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
			Name:      "Time (S)",
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
			Name:      "Avg Concurrent (QPS)",
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
			Name:      "Desired (Pods)",
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
		Name: "Desired Pods",
		Style: chart.Style{
			Show:        true,
			StrokeWidth: 3,
		},
	}
	runningPoints := chart.ContinuousSeries{
		Name:  "Running Pods",
		Style: chart.StyleShow(),
	}
	concurrentPoints := chart.ContinuousSeries{
		Name:  "Avg Concurrent QPS",
		Style: chart.StyleShow(),
	}

	return ch, desiredPoints, runningPoints, concurrentPoints
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
	} else if l.step < 80 {
		return 20
	}

	return 10
}

func (l *linear) RunningPods() int {
	if l.step < 100 {
		return 1
	} else if l.step < 350 {
		return 2
	} else if l.step < 550 {
		return 4
	} else if l.step < 650 {
		return 6
	} else if l.step < 750 {
		return 4
	}

	return 4
}

func (l *linear) Step(step int) {
	l.step = step
}

func makeEndpoints() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
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
	_, err := fakeClient.CoreV1().Endpoints(testNamespace).Create(ep)
	if err != nil {
		logger.Fatalf("could not create endpoint: %s", err.Error())
	}

	err = endpointsInformer.Informer().GetIndexer().Add(ep)
	if err != nil {
		logger.Fatalf("could not create endpoint informer: %s", err.Error())
	}
}

func updateEndpoints(ep *corev1.Endpoints) {
	_, err := fakeClient.CoreV1().Endpoints(testNamespace).Update(ep)
	if err != nil {
		logger.Fatalf("could not update endpoint: %s", err.Error())
	}

	err = endpointsInformer.Informer().GetIndexer().Update(ep)
	if err != nil {
		logger.Fatalf("could not update endpoint informer: %s", err.Error())
	}
}
