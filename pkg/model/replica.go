/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package model

import (
	"fmt"
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

func (rr *RevisionReplica) Run(startingAt time.Time) {
	rr.env.ListenForScheduling(rr.executable.name, finishLaunching, rr)
	rr.env.ListenForScheduling(rr.autoscaler.name, finishLaunchingReplica, rr)
	rr.env.ListenForScheduling(rr.autoscaler.name, terminateReplica, rr)

	rr.nextEvt = simulator.NewGeneralEvent(
		launchReplica,
		rr.env.Time().Add(10*time.Millisecond),
		rr,
	)
	rr.env.Schedule(rr.nextEvt)

	rr.executable.AddRevisionReplica(rr)
	rr.executable.Run(rr.nextEvt.OccursAt())

	rr.env.Schedule(simulator.NewGeneralEvent(
		terminateReplica,
		rr.env.Time().Add(8*time.Minute),
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
			event.OccursAt().Add(2*time.Second),
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
			fmt.Printf("%d %s\nfrom event: %#v\nwith subject: %s\nfrom state: %s to state: %s\n", event.OccursAt().UnixNano(), err.Error(), event, event.SubjectIdentity(), current, rr.fsm.Current())
		}
	}

	return simulator.StateTransitionResult{FromState: current, ToState: rr.fsm.Current()}
}

func (rr *RevisionReplica) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunching:
		rr.env.Schedule(simulator.NewGeneralEvent(
			finishLaunchingReplica,
			event.OccursAt().Add(10*time.Millisecond),
			rr,
		))
	case killProcess:
		rr.env.Schedule(simulator.NewGeneralEvent(
			finishTerminatingReplica,
			event.OccursAt().Add(10*time.Millisecond),
			rr,
		))
	}
}

func (rr *RevisionReplica) OnMovement(movement simulator.StockMovementEvent) (result simulator.MovementResult) {
	return simulator.MovementResult{
		FromStock: movement.From(),
		ToStock:   movement.To(),
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
