package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	StateRequestNotYetArrived      = "RequestNotYetArrived"
	StateRequestArrived            = "RequestArrived"
	StateRequestBuffered           = "RequestBuffered"
	StateRequestSentToReplica      = "RequestSentToReplica"
	StateRequestProcessing         = "RequestProcessing"
	StateRequestProcessingFinished = "RequestProcessingFinished"

	requestArrivedAtIngress = "request_arrived_at_ingress"
	requestBuffered         = "placed_request_in_buffer"
	sentRequestToReplica    = "sent_request_to_replica"
	beginRequestProcessing  = "begin_request_processing"
	finishRequestProcessing = "finish_request_processing"
)

var (
	evtRequestReceivedAtIngress = fsm.EventDesc{Name: requestArrivedAtIngress, Src: []string{StateRequestNotYetArrived}, Dst: StateRequestArrived}
	evtRequestedBuffered        = fsm.EventDesc{Name: requestBuffered, Src: []string{StateRequestArrived}, Dst: StateRequestBuffered}
	evtSentRequestToReplica     = fsm.EventDesc{Name: sentRequestToReplica, Src: []string{StateRequestArrived, StateRequestBuffered}, Dst: StateRequestSentToReplica}
	evtBeginRequestProcessing   = fsm.EventDesc{Name: beginRequestProcessing, Src: []string{StateRequestSentToReplica}, Dst: StateRequestProcessing}
	evtFinishRequestProcessing  = fsm.EventDesc{Name: finishRequestProcessing, Src: []string{StateRequestProcessing}, Dst: StateRequestProcessingFinished}
)

type Request struct {
	name        simulator.ProcessIdentity
	fsm         *fsm.FSM
	env         *simulator.Environment
	buffer      *KBuffer
	arrivalTime time.Time
}

func (r *Request) Identity() simulator.ProcessIdentity {
	return r.name
}

func (r *Request) OnOccurrence(event *simulator.Event) (result simulator.TransitionResult) {
	n := ""
	switch event.Name {
	case requestArrivedAtIngress:
		r.env.Schedule(&simulator.Event{
			OccursAt: event.OccursAt.Add(1 * time.Nanosecond),
			Name:     requestBuffered,
			Subject:  r,
		})
	case requestBuffered:
		r.buffer.AddRequest(r.name, r)
		// OnSchedule() for replica launched
	case sentRequestToReplica:
		r.buffer.DeleteRequest(r.name)

		r.env.Schedule(&simulator.Event{
			OccursAt: event.OccursAt.Add(10 * time.Millisecond),
			Name:     beginRequestProcessing,
			Subject:  r,
		})
	case beginRequestProcessing:
		rnd := rand.Intn(900) + 100

		r.env.Schedule(&simulator.Event{
			OccursAt: event.OccursAt.Add(time.Duration(rnd) * time.Millisecond), // TODO: function that respects utilisation
			Name:     finishRequestProcessing,
			Subject:  r,
		})
	case finishRequestProcessing:
		duration := event.OccursAt.Sub(r.arrivalTime)
		n = fmt.Sprintf("Request took %dms", duration.Nanoseconds()/1000000)
	}

	currentState := r.fsm.Current()
	err := r.fsm.Event(string(event.Name))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return simulator.TransitionResult{FromState: currentState, ToState: r.fsm.Current(), Note: n}
}

func (r *Request) Run() {
	r.env.Schedule(&simulator.Event{
		OccursAt: r.arrivalTime,
		Name:     requestArrivedAtIngress,
		Subject:  r,
	})
}

func NewRequest(env *simulator.Environment, buffer *KBuffer, arrivalTime time.Time) *Request {
	req := &Request{
		name:        simulator.ProcessIdentity(fmt.Sprintf("req-%012d", rand.Int63n(100000000000))),
		env:         env,
		buffer:      buffer,
		arrivalTime: arrivalTime,
	}
	req.fsm = fsm.NewFSM(
		StateRequestNotYetArrived,
		fsm.Events{
			evtRequestReceivedAtIngress,
			evtRequestedBuffered,
			evtSentRequestToReplica,
			evtBeginRequestProcessing,
			evtFinishRequestProcessing,
		},
		fsm.Callbacks{},
	)

	buffer.AddRequest(req.name, req)

	return req
}
