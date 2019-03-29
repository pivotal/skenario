package newmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/knative/pkg/logging"
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
	Model
	newsimulator.MovementListener
}

type knativeAutoscaler struct {
	env             newsimulator.Environment
	tickTock        *tickTock
	cluster         ClusterModel
	autoscaler      autoscaler.UniScaler
	ctx             context.Context
	lastDesired     int32
}

func (kas *knativeAutoscaler) Env() newsimulator.Environment {
	return kas.env
}

func (kas *knativeAutoscaler) OnMovement(movement newsimulator.Movement) error {
	switch movement.Kind() {
	case MvWaitingToCalculating:
		desired, ok := kas.autoscaler.Scale(kas.ctx, movement.OccursAt())
		if !ok {
			movement.AddNote("autoscaler.Scale() was unsuccessful")
		} else {
			if desired > kas.lastDesired {
				movement.AddNote(fmt.Sprintf("%d ⇑ %d", kas.lastDesired, desired))
			} else if desired < kas.lastDesired {
				movement.AddNote(fmt.Sprintf("%d ⥥ %d", kas.lastDesired, desired))
			}

			kas.lastDesired = desired
			kas.cluster.SetDesired(desired)
		}

		kas.env.AddToSchedule(newsimulator.NewMovement(MvCalculatingToWaiting, movement.OccursAt().Add(1*time.Nanosecond), kas.tickTock, kas.tickTock))
	case MvCalculatingToWaiting:
		kas.env.AddToSchedule(newsimulator.NewMovement(MvWaitingToCalculating, movement.OccursAt().Add(2*time.Second), kas.tickTock, kas.tickTock))
	}

	return nil
}

func NewKnativeAutoscaler(env newsimulator.Environment, startAt time.Time, cluster ClusterModel) KnativeAutoscaler {
	logger := newLogger()
	ctx := newLoggedCtx(logger)
	kpa := newKpa(logger)

	kas := &knativeAutoscaler{
		env:             env,
		tickTock:        &tickTock{},
		cluster:         cluster,
		autoscaler:      kpa,
		ctx:             ctx,
	}

	firstCalculation := newsimulator.NewMovement(MvWaitingToCalculating, startAt.Add(2001*time.Millisecond), kas.tickTock, kas.tickTock)
	firstCalculation.AddNote("First calculation")

	env.AddToSchedule(firstCalculation)
	err := env.AddMovementListener(kas)
	if err != nil {
		panic(err.Error())
	}

	return kas
}

func newLogger() *zap.SugaredLogger {
	devCfg := zap.NewDevelopmentConfig()
	devCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	devCfg.OutputPaths = []string{"stdout"}
	devCfg.ErrorOutputPaths = []string{"stderr"}
	unsugaredLogger, err := devCfg.Build()
	if err != nil {
		panic(err.Error())
	}
	return unsugaredLogger.Sugar()
}

func newLoggedCtx(logger *zap.SugaredLogger) context.Context {
	return logging.WithLogger(context.Background(), logger)
}

func newKpa(logger *zap.SugaredLogger) *autoscaler.Autoscaler {
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
