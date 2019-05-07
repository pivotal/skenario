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
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/simulator"
)

func TestReplicasDesired(t *testing.T) {
	spec.Run(t, "ReplicasDesired stock", testReplicasDesired, spec.Report(report.Terminal{}))
}

func testReplicasDesired(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicasDesiredStock
	var rawSubject *replicasDesiredStock
	var config ReplicasConfig
	var replicaSource ReplicaSource
	var replicasLaunching, replicasActive simulator.ThroughStock
	var replicasTerminated simulator.SinkStock
	var envFake *FakeEnvironment

	it.Before(func() {
		replicasLaunching = simulator.NewThroughStock("ReplicasLaunching", "Replica")
		replicasActive = simulator.NewThroughStock("ReplicasActive", "Replica")
		replicasTerminated = simulator.NewThroughStock("ReplicasTerminated", "Replica")
		replicaSource = NewReplicaSource(envFake, nil, nil)
		config = ReplicasConfig{LaunchDelay: 111 * time.Nanosecond, TerminateDelay: 222 * time.Nanosecond}
		envFake = new(FakeEnvironment)
		envFake.Movements = make([]simulator.Movement, 0)

		subject = NewReplicasDesiredStock(envFake, config, replicaSource, replicasLaunching, replicasActive, replicasTerminated)
		rawSubject = subject.(*replicasDesiredStock)
	})

	describe("Name()", func() {
		it("calls itself ReplicasDesired", func() {
			assert.Equal(t, simulator.StockName("ReplicasDesired"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Desireds", func() {
			assert.Equal(t, simulator.EntityKind("Desired"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("gives the count of added and removed", func() {
			assert.Equal(t, uint64(0), subject.Count())

			subject.Add(simulator.NewEntity("desired-1", "Desired"))
			assert.Equal(t, uint64(1), subject.Count())

			subject.Remove()
			assert.Equal(t, uint64(0), subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("returns an array of Desired", func() {
			ent := simulator.NewEntity("desired-1", "Desired")
			subject.Add(ent)
			assert.Equal(t, []*simulator.Entity{&ent}, subject.EntitiesInStock())
		})
	})

	describe("Add()", func() {
		it.Before(func() {
			subject.Add(simulator.NewEntity("add-1", "Desired"))
		})

		it("schedules movements of new entities from ReplicaSource to ReplicasLaunching", func() {
			assert.Equal(t, simulator.MovementKind("begin_launch"), envFake.Movements[0].Kind())
		})

		it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
			assert.Equal(t, simulator.MovementKind("finish_launching"), envFake.Movements[1].Kind())
		})

		it("adds the LaunchDelay to the launch time", func() {
			assert.Equal(t, envFake.TheTime.Add(111*time.Nanosecond), envFake.Movements[1].OccursAt())
		})
	})

	describe("Remove()", func() {
		it.Before(func() {
			rawSubject.delegate.Add(simulator.NewEntity("Removeable", "Desired"))
		})

		describe("there are launching replicas but no active replicas", func() {
			it.Before(func() {
				err := rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching", simulator.EntityKind("Replica")))
				assert.NoError(t, err)

				subject.Remove()
			})

			it("schedules movements from ReplicasLaunching to ReplicasTerminating", func() {
				assert.Len(t, envFake.Movements, 1)
				assert.Equal(t, simulator.MovementKind("terminate_launch"), envFake.Movements[0].Kind())
			})

			it("adds the TerminateDelay to the termination time", func() {
				assert.Equal(t, envFake.TheTime.Add(222*time.Nanosecond), envFake.Movements[0].OccursAt())
			})
		})

		describe("there are active replicas but no launching replicas", func() {
			it.Before(func() {
				newReplica := NewReplicaEntity(envFake, nil, nil, "1.2.1.2")
				err := rawSubject.replicasActive.Add(newReplica)
				assert.NoError(t, err)

				subject.Remove()
			})

			it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
				assert.Len(t, envFake.Movements, 1)
				assert.Equal(t, simulator.MovementKind("terminate_active"), envFake.Movements[0].Kind())
			})

			it("adds the TerminateDelay to the termination time", func() {
				assert.Equal(t, envFake.TheTime.Add(222*time.Nanosecond), envFake.Movements[0].OccursAt())
			})
		})

		//TODO: this won't work properly without batch movement: https://github.com/pivotal/skenario/issues/7
		describe.Pend("there is a mix of active and launching replicas", func() {
			it.Before(func() {
				newReplica := NewReplicaEntity(envFake, nil, nil, "3.4.3.4")
				err := rawSubject.replicasActive.Add(newReplica)
				assert.NoError(t, err)
				err = rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching", simulator.EntityKind("Replica")))
				assert.NoError(t, err)

				subject.Remove()
				subject.Remove()
			})

			it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
				assert.Len(t, envFake.Movements, 2)
				assert.Equal(t, "terminate_launch", string(envFake.Movements[0].Kind()))
				assert.Equal(t, "terminate_active", string(envFake.Movements[1].Kind()))
			})
		})

		describe.Pend("there are no active or launching replicas", func() {
			// can this actually happen?
		})
	})
}
