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
	name        string
	fsm         *fsm.FSM
	env         *simulator.Environment
	buffer      *KBuffer
	destination *RevisionReplica
	arrivalTime time.Time
}

func (r *Request) Advance(t time.Time, eventName string) (identifier, fromState, toState, note string) {
	n := ""
	switch eventName {
	case requestArrivedAtIngress:
		if r.destination.fsm.Is(StateReplicaActive) {
			r.env.Schedule(&simulator.Event{
				Time:        t.Add(1 * time.Nanosecond),
				EventName:   sentRequestToReplica,
				AdvanceFunc: r.Advance,
			})
		} else {
			r.env.Schedule(&simulator.Event{
				Time:        t.Add(1 * time.Nanosecond),
				EventName:   requestBuffered,
				AdvanceFunc: r.Advance,
			})
		}
	case requestBuffered:
		r.buffer.AddRequest(r.name, r)

		if r.destination.nextEvt.EventName == finishLaunchingReplica {
			r.env.Schedule(&simulator.Event{
				Time:        r.destination.nextEvt.Time.Add(10 * time.Millisecond),
				EventName:   sentRequestToReplica,
				AdvanceFunc: r.Advance,
			})
		}
	case sentRequestToReplica:
		r.buffer.DeleteRequest(r.name)

		r.env.Schedule(&simulator.Event{
			Time:        t.Add(10 * time.Millisecond),
			EventName:   beginRequestProcessing,
			AdvanceFunc: r.Advance,
		})
	case beginRequestProcessing:
		rnd := rand.Intn(900) + 100

		r.env.Schedule(&simulator.Event{
			Time:        t.Add(time.Duration(rnd) * time.Millisecond), // TODO: function that respects utilisation
			EventName:   finishRequestProcessing,
			AdvanceFunc: r.Advance,
		})
	case finishRequestProcessing:
		duration := t.Sub(r.arrivalTime)
		n = fmt.Sprintf("Request took %dms", duration.Nanoseconds()/1000000)
	}

	currentState := r.fsm.Current()
	err := r.fsm.Event(eventName)
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return r.name, currentState, r.fsm.Current(), n
}

func (r *Request) Run() {
	r.env.Schedule(&simulator.Event{
		Time:        r.arrivalTime,
		EventName:   requestArrivedAtIngress,
		AdvanceFunc: r.Advance,
	})
}

func NewRequest(env *simulator.Environment, buffer *KBuffer, destination *RevisionReplica, arrivalTime time.Time) *Request {
	req := &Request{
		name:        fmt.Sprintf("req-%012d", rand.Int63n(100000000000)),
		env:         env,
		buffer:      buffer,
		destination: destination,
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
