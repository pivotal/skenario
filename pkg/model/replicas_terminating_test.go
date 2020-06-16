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
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"skenario/pkg/simulator"
	"testing"
	"time"
)

func TestReplicasTerminating(t *testing.T) {
	spec.Run(t, "ReplicasTerminating stock", testReplicasTerminating, spec.Report(report.Terminal{}))
}

func testReplicasTerminating(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicasTerminatingStock
	var envFake *FakeEnvironment
	var replicaFake *FakeReplica
	var config ReplicasConfig
	var processingStock RequestsProcessingStock
	var terminatedStock simulator.SinkStock

	it.Before(func() {
		envFake = new(FakeEnvironment)
		replicaFake = new(FakeReplica)
		config = ReplicasConfig{LaunchDelay: 111 * time.Nanosecond, TerminateDelay: 222 * time.Nanosecond}
		terminatedStock = simulator.NewSinkStock("ReplicasTerminated", "Replica")
		subject = NewReplicasTerminatingStock(envFake, config, terminatedStock)
	})

	describe("Name()", func() {
		it("calls itself ReplicasDesired", func() {
			assert.Equal(t, simulator.StockName("ReplicasTerminating"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Replicas", func() {
			assert.Equal(t, simulator.EntityKind("Replica"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("gives the count of added and removed", func() {
			assert.Equal(t, uint64(0), subject.Count())

			subject.Add(replicaFake)
			assert.Equal(t, uint64(1), subject.Count())

			subject.Remove()
			assert.Equal(t, uint64(0), subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("returns an array of Replicas", func() {
			subject.Add(replicaFake)
			var castFake simulator.Entity
			castFake = replicaFake
			assert.Equal(t, []*simulator.Entity{&castFake}, subject.EntitiesInStock())
		})
	})

	describe("Add()", func() {
		describe("when the replica has no requests processing", func() {
			it.Before(func() {
				subject.Add(replicaFake)
			})

			it("schedules movements from ReplicasTerminating to ReplicasTerminated", func() {
				assert.Len(t, envFake.Movements, 1)
				assert.Equal(t, simulator.MovementKind("finish_terminating"), envFake.Movements[0].Kind())
			})

			it("schedules movements that occur after TerminateDelay", func() {
				assert.Equal(t, envFake.TheTime.Add(222*time.Nanosecond), envFake.Movements[0].OccursAt())
			})
		})

		describe.Focus("when the replica has requests processing", func() {
			it.Before(func() {
				totalCPUCapacityMillisPerSecond := 100.0
				occupiedCPUCapacityMillisPerSecond := 0.0
				failedSink := simulator.NewSinkStock("RequestsFailed", "Request")
				processingStock = NewRequestsProcessingStock(envFake, 111, simulator.NewSinkStock("RequestsCompleted", "Request"),
					&failedSink, &totalCPUCapacityMillisPerSecond, &occupiedCPUCapacityMillisPerSecond)
				bufferStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(), nil)
				err := processingStock.Add(NewRequestEntity(envFake, bufferStock, RequestConfig{CPUTimeMillis: 500, IOTimeMillis: 500, Timeout: 1 * time.Second}))
				require.NoError(t, err)
				replicaFake.ProcessingStock = processingStock
				err = subject.Add(replicaFake)
				require.NoError(t, err)
			})

			it("schedules movements from ReplicasTerminating to ReplicasTerminated", func() {
				assert.Len(t, envFake.Movements, 2) // 1 request move and 1 replica move
				assert.Equal(t, simulator.MovementKind("finish_terminating"), envFake.Movements[1].Kind())
			})

			it("schedules movements that occur after the last request completes + TerminateDelay", func() {
				requestTimePlusDelayTime := envFake.TheTime.Add(222 * time.Nanosecond).Add(1 * time.Second)
				assert.Equal(t, requestTimePlusDelayTime, envFake.Movements[1].OccursAt())
			})
		})
	})
}
