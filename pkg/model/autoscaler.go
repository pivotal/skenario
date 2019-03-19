package model

import (
	"context"
	"fmt"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/looplab/fsm"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"knative-simulator/pkg/simulator"
)

const (
	StateAutoscalerWaiting     = "AutoscalerWaiting"
	StateAutoscalerCalculating = "AutoscalerCalculating"

	waitForNextCalculation = "wait_for_next_calculation"
	calculateScale         = "calculate_scale"

	tickInterval           = 2 * time.Second
	stableWindow           = 60 * time.Second
	panicWindow            = 6 * time.Second
	scaleToZeroGracePeriod = 30 * time.Second
	targetConcurrency      = 5.0
	maxScaleUpRate         = 10.0
	testNamespace          = "simulator-namespace"
	testName               = "revisionService"
)

var logger *zap.SugaredLogger

type KnativeAutoscaler struct {
	name       simulator.ProcessIdentity
	fsm        *fsm.FSM
	env        *simulator.Environment
	autoscaler *autoscaler.Autoscaler
	replicas   []*RevisionReplica
	exec       *Executable
	endpoints  *ReplicaEndpoints
}

func (ka *KnativeAutoscaler) Identity() simulator.ProcessIdentity {
	return ka.name
}

func (ka *KnativeAutoscaler) OnOccurrence(event *simulator.Event) (result simulator.TransitionResult) {
	n := ""

	switch event.Name {
	case waitForNextCalculation:
		// could be extracted to setup in a loop
		ka.env.Schedule(&simulator.Event{
			Name:     calculateScale,
			OccursAt: event.OccursAt.Add(tickInterval),
			Subject:  nil,
		})
	case calculateScale:
		for _, rr := range ka.replicas {
			stat := autoscaler.Stat{
				Time:                      &event.OccursAt,
				PodName:                   string(rr.name),
				AverageConcurrentRequests: 10,
				RequestCount:              10,
			}
			ka.autoscaler.Record(context.Background(), stat)
		}

		currentReplicas := int32(len(ka.replicas))
		desiredScale, ok := ka.autoscaler.Scale(context.Background(), event.OccursAt)
		if ok {
			if desiredScale > currentReplicas {
				gap := desiredScale - currentReplicas
				for i := int32(0); i < gap; i++ {
					r := NewRevisionReplica(
						simulator.ProcessIdentity(fmt.Sprintf("replica-%d", i)),
						ka.exec,
						ka.endpoints,
						ka.env,
					)

					ka.replicas = append(ka.replicas, r)
					r.Run()
				}

				n = fmt.Sprintf("Scaling up from %d to %d", desiredScale, currentReplicas)
			} else if desiredScale < currentReplicas {
				gap := currentReplicas - desiredScale
				for i := int32(0); i < gap; i++ {
					r := ka.replicas[i]
					ka.env.Schedule(&simulator.Event{
						Name:     terminateReplica,
						OccursAt: event.OccursAt.Add(10 * time.Millisecond),
						Subject:  r,
					})
				}

				ka.replicas = ka.replicas[len(ka.replicas)-int(gap):]

				n = fmt.Sprintf("Scaling down to %d to %d", desiredScale, currentReplicas)
			}
		} else {
			n = "There was an error in scaling"
		}
	}

	currentState := ka.fsm.Current()
	err := ka.fsm.Event(string(event.Name))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return simulator.TransitionResult{FromState: currentState, ToState: ka.fsm.Current(), Note: n}
}

func NewAutoscaler(name simulator.ProcessIdentity, env *simulator.Environment, exec *Executable, endpoints *ReplicaEndpoints, kubernetesClient kubernetes.Interface) *KnativeAutoscaler {
	unsugaredLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err.Error())
	}
	logger = unsugaredLogger.Sugar()

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

	informerFactory := informers.NewSharedInformerFactory(kubernetesClient, 0)
	endpointsInformer := informerFactory.Core().V1().Endpoints()

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		testName,
		endpointsInformer,
		targetConcurrency,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	ka := &KnativeAutoscaler{
		name:       name,
		env:        env,
		autoscaler: as,
		exec:       exec,
		endpoints:  endpoints,
		replicas:   make([]*RevisionReplica, 0),
	}

	ka.fsm = fsm.NewFSM(
		StateAutoscalerWaiting,
		fsm.Events{
			fsm.EventDesc{Name: waitForNextCalculation, Src: []string{StateAutoscalerCalculating}, Dst: StateAutoscalerWaiting},
			fsm.EventDesc{Name: calculateScale, Src: []string{StateAutoscalerWaiting}, Dst: StateAutoscalerCalculating},
		},
		fsm.Callbacks{},
	)

	return ka
}
