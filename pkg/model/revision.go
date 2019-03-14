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
	name               string
	fsm                *fsm.FSM
	env                *simulator.Environment
	executable         *Executable
	numCurrentRequests int64
}

func (rr *RevisionReplica) Run() {
	r := rand.Intn(1000)

	rr.env.Schedule(&simulator.Event{
		Time:        rr.env.Time().Add(time.Duration(r) * time.Millisecond),
		EventName:   launchReplica,
		AdvanceFunc: rr.Advance,
	})
}

func (rr *RevisionReplica) Advance(t time.Time, eventName string) (identifier, outcome string) {
	// special cases
	switch eventName {
	case launchReplica:
		rr.executable.AddRevisionReplica(rr)
		rr.executable.Run(rr.env)
	case finishLaunchingReplica:
		rr.env.Schedule(&simulator.Event{
			Time:        t.Add(180 * time.Second),
			EventName:   terminateReplica,
			AdvanceFunc: rr.Advance,
		})
	case receiveRequest:
		rr.numCurrentRequests++

		rr.env.Schedule(&simulator.Event{
			Time:        t.Add(1 * time.Second),
			EventName:   completeRequest,
			AdvanceFunc: rr.Advance,
		})
	case completeRequest:
		rr.numCurrentRequests--
	case terminateReplica:
		rr.env.Schedule(&simulator.Event{
			Time:        t.Add(2 * time.Second),
			EventName:   killProcess,
			AdvanceFunc: rr.executable.Advance,
		})
	}

	current := rr.fsm.Current()
	err := rr.fsm.Event(eventName)
	if err != nil {
		panic(err.Error())
	}

	return rr.name, fmt.Sprintf("%s --> %s", current, rr.fsm.Current())
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
		fsm.Callbacks{
			"before_become_idle": func(e *fsm.Event) {
				if rr.numCurrentRequests > 0 {
					e.Cancel(nil)
				}
			},
		},
	)

	return rr
}
