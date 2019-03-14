package model

import (
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	StateReplicaNotLaunched = "ReplicaNotLaunched"
	StateReplicaLaunching   = "ReplicaLaunching"
	StateReplicaActive      = "ReplicaActive"
	StateReplicaTerminating = "ReplicaTerminating"
	StateReplicaTerminated  = "ReplicaTerminated"

	launchReplica            = "launch_replica"
	finishLaunchingReplica   = "finish_launching_replica"
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

	return rr.name, current, rr.fsm.Current(), ""
}

func NewRevisionReplica(name string, exec *Executable, env *simulator.Environment) *RevisionReplica {
	rr := &RevisionReplica{
		name:               name,
		env:                env,
		executable:         exec,
		numCurrentRequests: 0,
	}

	rr.fsm = fsm.NewFSM(
		StateReplicaNotLaunched,
		fsm.Events{
			{Name: launchReplica, Src: []string{StateReplicaNotLaunched}, Dst: StateReplicaLaunching},     // register callback with Executable
			{Name: finishLaunchingReplica, Src: []string{StateReplicaLaunching}, Dst: StateReplicaActive}, // register callback with Executable
			{Name: terminateReplica, Src: []string{StateReplicaActive}, Dst: StateReplicaTerminating},          // kill Executable as well?
			{Name: finishTerminatingReplica, Src: []string{StateReplicaTerminating}, Dst: StateReplicaTerminated}, // kill Executable as well?
		},
		fsm.Callbacks{},
	)

	return rr
}
