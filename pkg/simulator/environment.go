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

package simulator

import (
	"context"
	"fmt"
	"time"

	"skenario/pkg/plugin"
)

const (
	OccursInPast                            = "ScheduledToOccurInPast"
	OccursAfterHalt                         = "ScheduledToOccurAfterHalt"
	OccursSimultaneouslyWithAnotherMovement = "ScheduleCollidesWithAnotherMovement"
	FromStockIsEmpty                        = "FromStockEmptyAtMovementTime"
)

type Environment interface {
	Plugin() *plugin.PluginPartition
	AddToSchedule(movement Movement) (added bool)
	Run() (completed []CompletedMovement, ignored []IgnoredMovement, err error)
	CurrentMovementTime() time.Time
	HaltTime() time.Time
	Context() context.Context
	CPUUtilizations() []*CPUUtilization
	AppendCPUUtilization(cpuUtilization *CPUUtilization)
}

type CompletedMovement struct {
	Movement Movement
	Moved    Entity
}

type IgnoredMovement struct {
	Reason   string
	Movement Movement
	Moved    Entity
}

type CPUUtilization struct {
	CPUUtilization float64
	CalculatedAt   time.Time
}

type environment struct {
	ctx    context.Context
	plugin *plugin.PluginPartition

	current time.Time
	startAt time.Time
	haltAt  time.Time

	beforeScenario  ThroughStock
	runningScenario ThroughStock
	haltedScenario  ThroughStock

	futureMovements MovementPriorityQueue
	completed       []CompletedMovement
	ignored         []IgnoredMovement
	cpuUtilizations []*CPUUtilization
}

func (env *environment) Plugin() *plugin.PluginPartition {
	return env.plugin
}

func (env *environment) AddToSchedule(movement Movement) (added bool) {
	occursAfterCurrent := movement.OccursAt().After(env.current)
	occursBeforeHalt := movement.OccursAt().Before(env.haltAt)

	schedulable := occursAfterCurrent && occursBeforeHalt
	if schedulable {
		_, _, err := env.futureMovements.EnqueueMovement(movement)
		if err != nil {
			panic(fmt.Errorf("unknown error meant '%#v' was not added future movements: %s", movement, err.Error()))
		}
	} else if !occursAfterCurrent {
		env.ignored = append(env.ignored, IgnoredMovement{
			Reason:   OccursInPast,
			Movement: movement,
		})
	} else if !occursBeforeHalt {
		env.ignored = append(env.ignored, IgnoredMovement{
			Reason:   OccursAfterHalt,
			Movement: movement,
		})
	}

	return schedulable
}

func (env *environment) Run() ([]CompletedMovement, []IgnoredMovement, error) {
	for {
		var err error

		movement, err, closed := env.futureMovements.DequeueMovement()
		if err != nil {
			return nil, nil, err
		}

		if closed {
			break
		}

		env.current = movement.OccursAt()

		moved := movement.From().Remove()
		if moved == nil {
			env.ignored = append(env.ignored, IgnoredMovement{Movement: movement, Reason: FromStockIsEmpty})
		} else {
			movement.To().Add(moved)
			env.completed = append(env.completed, CompletedMovement{Movement: movement, Moved: moved})
		}
	}

	return env.completed, env.ignored, nil
}

func (env *environment) CurrentMovementTime() time.Time {
	return env.current
}

func (env *environment) HaltTime() time.Time {
	return env.haltAt
}

func (env *environment) Context() context.Context {
	return env.ctx
}

var environmentSequence int32 = 0

func (env *environment) CPUUtilizations() []*CPUUtilization {
	return env.cpuUtilizations
}

func (env *environment) AppendCPUUtilization(cpuUtilization *CPUUtilization) {
	env.cpuUtilizations = append(env.cpuUtilizations, cpuUtilization)
}

func NewEnvironment(ctx context.Context, startAt time.Time, runFor time.Duration) Environment {
	pqueue := NewMovementPriorityQueue()
	return newEnvironment(ctx, startAt, runFor, pqueue)
}

func newEnvironment(ctx context.Context, startAt time.Time, runFor time.Duration, pqueue MovementPriorityQueue) *environment {
	beforeStock := NewThroughStock("BeforeScenario", "Scenario")
	runningStock := NewThroughStock("RunningScenario", "Scenario")
	haltingStock := NewHaltingSink("HaltedScenario", "Scenario", pqueue)

	env := &environment{
		ctx:     ctx,
		plugin:  plugin.NewPluginPartition(),
		startAt: startAt,
		haltAt:  startAt.Add(runFor).Add(1 * time.Nanosecond), // make temporary space for the Halt Scenario movement
		current: startAt.Add(-1 * time.Nanosecond),            // make temporary space for the Start Scenario movement

		beforeScenario:  beforeStock,
		runningScenario: runningStock,
		haltedScenario:  haltingStock,
		futureMovements: pqueue,
		completed:       make([]CompletedMovement, 0),
		ignored:         make([]IgnoredMovement, 0),
		cpuUtilizations: make([]*CPUUtilization, 0),
	}

	env = setupScenarioMovements(env, startAt, env.haltAt.Add(-1*time.Nanosecond), env.beforeScenario, env.runningScenario, env.haltedScenario)
	env.current = startAt // restore proper starting time
	env.haltAt = env.haltAt.Add(-1 * time.Nanosecond)

	return env
}

func setupScenarioMovements(env *environment, startAt time.Time, haltAt time.Time, beforeScenario, runningScenario, haltedScenario ThroughStock) *environment {
	scenarioEntity := NewEntity("Scenario", "Scenario")
	err := beforeScenario.Add(scenarioEntity)
	if err != nil {
		panic(fmt.Errorf("could not add Scenario entity to haltedScenario: %s", err.Error()))
	}

	startMovement := NewMovement("start_to_running", startAt, beforeScenario, runningScenario)
	startMovement.AddNote("Start scenario")
	haltMovement := NewMovement("running_to_halted", haltAt, runningScenario, haltedScenario)
	haltMovement.AddNote("Halt scenario")

	env.AddToSchedule(startMovement)
	env.AddToSchedule(haltMovement)

	return env
}
