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
	name       simulator.ProcessIdentity
	fsm        *fsm.FSM
	env        *simulator.Environment
	executable *Executable
	nextEvt    simulator.Event
	autoscaler *KnativeAutoscaler
}

func (rr *RevisionReplica) Run() {
	r := rand.Intn(1000)

	if !rr.executable.fsm.Is(StateReplicaActive) {
		rr.env.ListenForScheduling(rr.executable.name, finishLaunching, rr)
		rr.env.ListenForScheduling(rr.autoscaler.name, finishLaunchingReplica, rr)
		rr.env.ListenForScheduling(rr.autoscaler.name, terminateReplica, rr)
	}

	rr.nextEvt = simulator.NewGeneralEvent(
		launchReplica,
		rr.env.Time().Add(time.Duration(r) * time.Millisecond),
		rr,
	)
	rr.env.Schedule(rr.nextEvt)

	rr.executable.AddRevisionReplica(rr)
	rr.executable.Run(rr.nextEvt.OccursAt())

	rr.env.Schedule(simulator.NewGeneralEvent(
		terminateReplica,
		rr.env.Time().Add(8 * time.Minute),
		rr,
	))
}

func (rr *RevisionReplica) Identity() simulator.ProcessIdentity {
	return rr.name
}

func (rr *RevisionReplica) OnOccurrence(event simulator.Event) (result simulator.StateTransitionResult) {
	currEventTime := rr.nextEvt.OccursAt()

	switch event.Name() {
	case terminateReplica:
		rr.nextEvt = simulator.NewGeneralEvent(
			finishTerminatingReplica,
			event.OccursAt().Add(2 * time.Second),
			rr,
		)
	}

	if rr.nextEvt.OccursAt().After(currEventTime) {
		rr.env.Schedule(rr.nextEvt)
	}

	current := rr.fsm.Current()
	err := rr.fsm.Event(string(event.Name()))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return simulator.StateTransitionResult{FromState: current, ToState: rr.fsm.Current()}
}

func (rr *RevisionReplica) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunching:
		rr.env.Schedule(simulator.NewGeneralEvent(
			finishLaunchingReplica,
			event.OccursAt().Add(10 * time.Millisecond),
			rr,
		))
	case killProcess:
		rr.env.Schedule(simulator.NewGeneralEvent(
			finishTerminatingReplica,
			event.OccursAt().Add(10 * time.Millisecond),
			rr,
		))
	}
}

func (rr *RevisionReplica) UpdateStock(movement simulator.StockMovementEvent) {
	// do nothing
}

func NewRevisionReplica(name simulator.ProcessIdentity, exec *Executable, env *simulator.Environment, autoscaler *KnativeAutoscaler) *RevisionReplica {
	rr := &RevisionReplica{
		name:       name,
		env:        env,
		executable: exec,
		autoscaler: autoscaler,
	}

	rr.fsm = fsm.NewFSM(
		StateReplicaNotLaunched,
		fsm.Events{
			{Name: launchReplica, Src: []string{StateReplicaNotLaunched}, Dst: StateReplicaLaunching},             // register callback with Executable
			{Name: finishLaunchingReplica, Src: []string{StateReplicaLaunching}, Dst: StateReplicaActive},         // register callback with Executable
			{Name: terminateReplica, Src: []string{StateReplicaActive}, Dst: StateReplicaTerminating},             // kill Executable as well?
			{Name: finishTerminatingReplica, Src: []string{StateReplicaTerminating}, Dst: StateReplicaTerminated}, // kill Executable as well?
		},
		fsm.Callbacks{},
	)

	return rr
}
