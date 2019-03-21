package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	StateRequestArrived            = "RequestArrived"
	StateRequestBuffered           = "RequestBuffered"
	StateRequestProcessing         = "RequestProcessing"
	StateRequestProcessingFinished = "RequestFinished"

	bufferRequest           = "buffer_request"
	beginRequestProcessing  = "begin_request_processing"
	finishRequestProcessing = "finish_request_processing"
)

var (
	evtRequestedBuffered       = fsm.EventDesc{Name: bufferRequest, Src: []string{StateRequestArrived}, Dst: StateRequestBuffered}
	evtBeginRequestProcessing  = fsm.EventDesc{Name: beginRequestProcessing, Src: []string{StateRequestBuffered}, Dst: StateRequestProcessing}
	evtFinishRequestProcessing = fsm.EventDesc{Name: finishRequestProcessing, Src: []string{StateRequestProcessing}, Dst: StateRequestProcessingFinished}
)

type Request struct {
	name         simulator.ProcessIdentity
	fsm          *fsm.FSM
	env          *simulator.Environment
	currentStock simulator.Stock
	arrivalTime  time.Time
}

func (r *Request) Identity() simulator.ProcessIdentity {
	return r.name
}

func (r *Request) OnOccurrence(event simulator.Event) (result simulator.StateTransitionResult) {
	n := ""

	switch event.Name() {
	case bufferRequest:
		n = "bu bu buffer"
		// do nothing
	case beginRequestProcessing:
		n = "lolwut"
		// do nothing
	case finishRequestProcessing:
		duration := event.OccursAt().Sub(r.arrivalTime)
		n = fmt.Sprintf("Request took %dms", duration.Nanoseconds()/1000000)
	}

	currentState := r.fsm.Current()
	err := r.fsm.Event(string(event.Name()))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return simulator.StateTransitionResult{FromState: currentState, ToState: r.fsm.Current(), Note: n}
}

func (r *Request) OnMovement(movement simulator.StockMovementEvent) (result simulator.MovementResult) {
	r.currentStock = movement.To()

	beginProc := simulator.NewGeneralEvent(
		beginRequestProcessing,
		movement.OccursAt().Add(10*time.Microsecond),
		r,
	)
	r.env.Schedule(beginProc)

	finishProc := simulator.NewGeneralEvent(
		finishRequestProcessing,
		movement.OccursAt().Add(500*time.Millisecond),
		r,
	)
	r.env.Schedule(finishProc)

	return simulator.MovementResult{FromStock: movement.From(), ToStock: movement.To()}
}

func (r *Request) CurrentlyAt() simulator.Stock {
	return r.currentStock
}

func NewRequest(env *simulator.Environment, buffer *KBuffer, arrivalTime time.Time) *Request {
	req := &Request{
		name:         simulator.ProcessIdentity(fmt.Sprintf("req-%012d", rand.Int63n(100000000000))),
		env:          env,
		currentStock: buffer,
		arrivalTime:  arrivalTime,
	}

	req.fsm = fsm.NewFSM(
		StateRequestArrived,
		fsm.Events{
			evtRequestedBuffered,
			evtBeginRequestProcessing,
			evtFinishRequestProcessing,
		},
		fsm.Callbacks{},
	)

	env.Schedule(simulator.NewGeneralEvent(
		bufferRequest,
		arrivalTime.Add(1*time.Nanosecond),
		req,
	))

	return req
}
