package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	launchReplica            = "launch_replica"
	finishLaunchingReplica   = "finish_launching_replica"
	receiveRequest           = "receive_request"
	completeRequest          = "complete_request"
	terminateReplica         = "terminate_replica"
	finishTerminatingReplica = "finish_terminating_replica"
)

type RevisionReplica struct {
	name                string
	fsm                 *fsm.FSM
	env                 *simulator.Environment
	executable          *Executable
	numCurrentRequests  int64
	numBufferedRequests int64
	nextEvt             *simulator.Event
}

func (rr *RevisionReplica) Run() {
	r := rand.Intn(1000)

	rr.nextEvt = &simulator.Event{
		Time:        rr.env.Time().Add(time.Duration(r) * time.Millisecond),
		EventName:   launchReplica,
		AdvanceFunc: rr.Advance,
	}
	rr.env.Schedule(rr.nextEvt)

	rr.executable.AddRevisionReplica(rr)
	rr.executable.Run(rr.env, rr.nextEvt.Time)
}

func (rr *RevisionReplica) Advance(t time.Time, eventName string) (identifier, fromState, toState, note string) {
	currEventTime := rr.nextEvt.Time

	switch eventName {
	case launchReplica:
		// handled by Run
	case finishLaunchingReplica:
		// handled by the Executable
	case receiveRequest:
		switch rr.fsm.Current() {
		case "ReplicaNotLaunched":
		case "ReplicaLaunching":
			rr.numBufferedRequests++
			if rr.nextEvt.EventName == finishLaunchingReplica {
				rr.env.Schedule(&simulator.Event{
					Time:        currEventTime.Add(50 * time.Millisecond),
					EventName:   receiveRequest,
					AdvanceFunc: rr.Advance,
				})
			}

			return rr.name, "ReplicaLaunching", "ReplicaLaunching", fmt.Sprintf("numBufferedRequests: %3d currentNumRequests: %3d", rr.numBufferedRequests, rr.numCurrentRequests)
		case "ReplicaActive":
			rr.numCurrentRequests++
			if rr.numBufferedRequests > 0 {
				rr.numBufferedRequests--
			}

			rr.nextEvt = &simulator.Event{
				Time:        t.Add(1 * time.Second),
				EventName:   completeRequest,
				AdvanceFunc: rr.Advance,
			}
		}
	case completeRequest:
		rr.numCurrentRequests--
	case terminateReplica:
		rr.nextEvt = &simulator.Event{
			Time:        t.Add(2 * time.Second),
			EventName:   killProcess,
			AdvanceFunc: rr.executable.Advance,
		}
	}

	if rr.nextEvt.Time.After(currEventTime) {
		rr.env.Schedule(rr.nextEvt)
	}

	current := rr.fsm.Current()
	err := rr.fsm.Event(eventName)
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return rr.name, current, rr.fsm.Current(), fmt.Sprintf("numBufferedRequests: %3d currentNumRequests: %3d", rr.numBufferedRequests, rr.numCurrentRequests)
}

func NewRevisionReplica(name string, exec *Executable, env *simulator.Environment) *RevisionReplica {
	rr := &RevisionReplica{
		name:               name,
		env:                env,
		executable:         exec,
		numCurrentRequests: 0,
	}

	rr.fsm = fsm.NewFSM(
		"ReplicaNotLaunched",
		fsm.Events{
			{Name: launchReplica, Src: []string{"ReplicaNotLaunched"}, Dst: "ReplicaLaunching"},     // register callback with Executable
			{Name: finishLaunchingReplica, Src: []string{"ReplicaLaunching"}, Dst: "ReplicaActive"}, // register callback with Executable
			{Name: receiveRequest, Src: []string{"ReplicaActive"}, Dst: "ReplicaActive"},
			{Name: completeRequest, Src: []string{"ReplicaActive"}, Dst: "ReplicaActive"},
			{Name: terminateReplica, Src: []string{"ReplicaActive"}, Dst: "ReplicaTerminating"},             // kill Executable as well?
			{Name: finishTerminatingReplica, Src: []string{"ReplicaTerminating"}, Dst: "ReplicaTerminated"}, // kill Executable as well?
		},
		fsm.Callbacks{},
	)

	return rr
}
