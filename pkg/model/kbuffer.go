package model

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/knative/serving/pkg/autoscaler"

	"knative-simulator/pkg/simulator"
)

type KBuffer struct {
	env              *simulator.Environment
	requestsBuffered map[simulator.ProcessIdentity]*Request
	replicas         map[simulator.ProcessIdentity]*RevisionReplica
	autoscaler       *KnativeAutoscaler
}

const (
	addReplicaToKBuffer      = "add_replica_to_kbuffer"
	removeReplicaFromKBuffer = "remove_replica_from_kbuffer"
)

func (kb *KBuffer) Identity() simulator.ProcessIdentity {
	return "KBuffer"
}
func (kb *KBuffer) UpdateStock(movement simulator.StockMovementEvent) {
	switch movement.Name() {
	case bufferRequest:
	}
	if kb == movement.To() {
		numRequests := len(kb.requestsBuffered)
		numReplicas := len(kb.replicas)

		req := movement.Subject().(*Request)
		kb.requestsBuffered[req.Identity()] = req
		numRequests++

		if numRequests > 0 && numReplicas > 0 {
			var replicaKeys []simulator.ProcessIdentity
			for k := range kb.replicas {
				replicaKeys = append(replicaKeys, k)
			}
			key := replicaKeys[rand.Intn(numReplicas)]

			i := 1
			for _, v := range kb.requestsBuffered {
				mv := simulator.NewMovementEvent(
					sendToReplica,
					movement.OccursAt().Add(time.Duration(i)*time.Nanosecond),
					v,
					kb,
					kb.replicas[key],
				)

				kb.env.Schedule(mv)
				i++
			}
		} else if numRequests > 0 {
			occurs := movement.OccursAt().Add(1 * time.Nanosecond)
			kb.autoscaler.autoscaler.Record(context.Background(), autoscaler.Stat{
				Time:                      &occurs,
				PodName:                   "KBuffer",
				AverageConcurrentRequests: float64(numRequests),
				RequestCount:              int32(numRequests),
			})
		}

	} else if kb == movement.From() {
		delete(kb.requestsBuffered, movement.SubjectIdentity())
	} else {
		panic(fmt.Errorf("impossible movement: %+v", movement))
	}
}

func (kb *KBuffer) OnOccurrence(event simulator.Event) (result simulator.StateTransitionResult) {
	switch event.Name() {
	case addReplicaToKBuffer:
		// lol no generics
		gevt := event.(simulator.GeneralEvent)
		rr := gevt.Subject().(*RevisionReplica)

		kb.replicas[event.SubjectIdentity()] = rr
	case removeReplicaFromKBuffer:
		delete(kb.replicas, event.SubjectIdentity())
	}

	return simulator.StateTransitionResult{
		FromState: "KBufferActive",
		ToState:   "KBufferActive",
	}
}

func (kb *KBuffer) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunchingReplica:
		gevt := event.(simulator.GeneralEvent)
		rr := gevt.Subject().(*RevisionReplica)

		kb.env.Schedule(simulator.NewGeneralEvent(
			addReplicaToKBuffer,
			event.OccursAt().Add(10*time.Millisecond),
			rr,
		))
	case terminateReplica:
		gevt := event.(simulator.GeneralEvent)
		rr := gevt.Subject().(*RevisionReplica)

		kb.env.Schedule(simulator.NewGeneralEvent(
			removeReplicaFromKBuffer,
			event.OccursAt().Add(-10*time.Millisecond),
			rr,
		))
	}
}

func NewKBuffer(env *simulator.Environment, autoscaler *KnativeAutoscaler) *KBuffer {
	kb := &KBuffer{
		env:              env,
		requestsBuffered: make(map[simulator.ProcessIdentity]*Request),
		replicas:         make(map[simulator.ProcessIdentity]*RevisionReplica),
		autoscaler:       autoscaler,
	}

	return kb
}
