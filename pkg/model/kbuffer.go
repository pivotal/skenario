package model

import (
	"time"

	"knative-simulator/pkg/simulator"
)

type KBuffer struct {
	env              *simulator.Environment
	requestsBuffered map[simulator.ProcessIdentity]*Request
	replicas         map[simulator.ProcessIdentity]*RevisionReplica
}

func (kb *KBuffer) Identity() simulator.ProcessIdentity {
	return "KBuffer"
}

func (kb *KBuffer) AddStock(item simulator.Stockable) {
	req := item.(*Request)
	kb.requestsBuffered[req.Identity()] = req
}

func (kb *KBuffer) RemoveStock(item simulator.Stockable) {
	req := item.(*Request)
	delete(kb.requestsBuffered, req.Identity())
}

func (kb *KBuffer) OnSchedule(event simulator.Event) {
	// lol no generics
	gevt := event.(simulator.GeneralEvent)
	rr := gevt.Subject().(*RevisionReplica)

	switch event.Name() {
	case finishLaunchingReplica:
		kb.replicas[event.SubjectIdentity()] = rr
	case bufferRequest:
		// if there are replicas, send them
		// if not, buffer the requests
	}

	numRequests := len(kb.requestsBuffered)
	numReplicas := len(kb.replicas)

	if numRequests > 0 && numReplicas > 0 {
		i := time.Duration(1)
		for _, v := range kb.requestsBuffered {
			mv := simulator.NewMovementEvent(
				"kbuffer_to_replica",
				event.OccursAt().Add(i*time.Nanosecond),
				v,
				kb,
				rr,
			)

			kb.env.Schedule(mv)
			i++
		}
	}

}

func NewKBuffer(env *simulator.Environment) *KBuffer {
	return &KBuffer{
		env:              env,
		requestsBuffered: make(map[simulator.ProcessIdentity]*Request),
		replicas:         make(map[simulator.ProcessIdentity]*RevisionReplica),
	}
}
