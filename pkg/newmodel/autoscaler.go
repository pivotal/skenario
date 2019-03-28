package newmodel

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/informers"

	"knative-simulator/pkg/newsimulator"

	"github.com/knative/serving/pkg/autoscaler"
	fakes "k8s.io/client-go/kubernetes/fake"
)

const (
	MvWaitingToCalculating newsimulator.MovementKind = "autoscaler_wait"
	MvCalculatingToWaiting newsimulator.MovementKind = "autoscaler_calc"

	stableWindow                = 60 * time.Second
	panicWindow                 = 6 * time.Second
	scaleToZeroGracePeriod      = 30 * time.Second
	targetConcurrencyDefault    = 2.0
	targetConcurrencyPercentage = 0.5
	maxScaleUpRate              = 10.0
	testNamespace               = "simulator-namespace"
	testName                    = "revisionService"
)

type KnativeAutoscaler interface {
	newsimulator.MovementListener
}

type knativeAutoscaler struct {
	env        newsimulator.Environment
	tickTock   *tickTock
	autoscaler *autoscaler.Autoscaler
}

func (kas *knativeAutoscaler) OnMovement(movement newsimulator.Movement) error {
	if movement.Kind() == MvWaitingToCalculating {
		waitingMovement := newsimulator.NewMovement(
			MvCalculatingToWaiting,
			movement.OccursAt().Add(1*time.Nanosecond),
			kas.tickTock,
			kas.tickTock,
			"",
		)

		calculatingMovement := newsimulator.NewMovement(
			MvWaitingToCalculating,
			movement.OccursAt().Add(2*time.Second),
			kas.tickTock,
			kas.tickTock,
			"",
		)

		kas.env.AddToSchedule(waitingMovement)
		kas.env.AddToSchedule(calculatingMovement)
	}

	return nil
}

func NewKnativeAutoscaler(env newsimulator.Environment, startAt time.Time) KnativeAutoscaler {
	kas := &knativeAutoscaler{
		env:        env,
		tickTock:   &tickTock{},
		autoscaler: newKpa(),
	}

	firstCalculation := newsimulator.NewMovement(
		MvWaitingToCalculating,
		startAt.Add(2001*time.Millisecond),
		kas.tickTock,
		kas.tickTock,
		"First calculation",
	)

	env.AddToSchedule(firstCalculation)
	err := env.AddMovementListener(kas)
	if err != nil {
		panic(err.Error())
	}

	return kas
}

func newKpa() *autoscaler.Autoscaler {
	devCfg := zap.NewDevelopmentConfig()
	devCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	devCfg.OutputPaths = []string{"stdout"}
	devCfg.ErrorOutputPaths = []string{"stderr"}
	unsugaredLogger, err := devCfg.Build()
	if err != nil {
		panic(err.Error())
	}
	logger := unsugaredLogger.Sugar()
	defer logger.Sync()

	//ctx := logging.WithLogger(context.Background(), logger)

	config := &autoscaler.Config{
		MaxScaleUpRate:                       maxScaleUpRate,
		StableWindow:                         stableWindow,
		PanicWindow:                          panicWindow,
		ScaleToZeroGracePeriod:               scaleToZeroGracePeriod,
		ContainerConcurrencyTargetPercentage: targetConcurrencyPercentage,
		ContainerConcurrencyTargetDefault:    targetConcurrencyDefault,
	}

	dynConfig := autoscaler.NewDynamicConfig(config, logger)

	statsReporter, err := autoscaler.NewStatsReporter(testNamespace, testName, "config-1", "revision-1")
	if err != nil {
		logger.Fatalf("could not create stats reporter: %s", err.Error())
	}

	fakeClient := fakes.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)
	endpointsInformer := informerFactory.Core().V1().Endpoints()

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		testName,
		endpointsInformer,
		targetConcurrencyDefault,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	return as
}

type tickTock struct {
	asEntity newsimulator.Entity
}

func (tt *tickTock) Name() newsimulator.StockName {
	return "Autoscaler ticktock"
}

func (tt *tickTock) KindStocked() newsimulator.EntityKind {
	return newsimulator.EntityKind("KnativeAutoscaler")
}

func (tt *tickTock) Count() uint64 {
	return 1
}

func (tt *tickTock) Remove() newsimulator.Entity {
	return tt.asEntity
}

func (tt *tickTock) Add(entity newsimulator.Entity) error {
	tt.asEntity = entity

	return nil
}
