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

func TestRequestsBuffered(t *testing.T) {
	spec.Run(t, "RequestsBuffered stock", testRequestsBuffered, spec.Report(report.Terminal{}))
}

func testRequestsBuffered(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestsBufferedStock
	var rawSubject *requestsBufferedStock
	var envFake *FakeEnvironment
	var replicaStock ReplicasActiveStock
	var requestsFailedStock simulator.SinkStock
	var replicaFake *fakeReplica

	it.Before(func() {
		requestsFailedStock = simulator.NewSinkStock("RequestsFailed", "Request")
	})

	describe("NewRequestsBufferedStock()", func() {
		it.Before(func() {
			envFake = new(FakeEnvironment)
			replicaStock = NewReplicasActiveStock()
			subject = NewRequestsBufferedStock(envFake, replicaStock, nil)
			rawSubject = subject.(*requestsBufferedStock)
		})

		it("creates a delegate ThroughStock", func() {
			assert.NotNil(t, rawSubject.delegate)
			assert.Equal(t, simulator.StockName("RequestsBuffered"), rawSubject.delegate.Name())
			assert.Equal(t, simulator.EntityKind("Request"), rawSubject.delegate.KindStocked())
		})
	})

	describe("Add()", func() {
		var request RequestEntity

		describe("there are multiple replicas available to serve multiple requests", func() {
			it.Before(func() {
				envFake = new(FakeEnvironment)

				replicaStock = NewReplicasActiveStock()

				replicaFake = new(fakeReplica)
				replicaFake.fakeReplicaNum = 11
				replicaStock.Add(replicaFake)

				replicaFake = new(fakeReplica)
				replicaFake.fakeReplicaNum = 22
				replicaStock.Add(replicaFake)

				replicaFake = new(fakeReplica)
				replicaFake.fakeReplicaNum = 33
				replicaStock.Add(replicaFake)

				subject = NewRequestsBufferedStock(envFake, replicaStock, requestsFailedStock)

				subject.Add(NewRequestEntity(envFake, subject))
				subject.Add(NewRequestEntity(envFake, subject))
				subject.Add(NewRequestEntity(envFake, subject))
			})

			it("assigns the Requests to Replicas using round robin", func() {
				first := envFake.Movements[1]
				second := envFake.Movements[2]

				assert.Equal(t, simulator.MovementKind("send_to_replica"), first.Kind())
				assert.Equal(t, simulator.MovementKind("send_to_replica"), second.Kind())
				assert.NotEqual(t, first.To(), second.To())
			})
		})

		describe("there are no other requests yet", func() {
			describe("there is at least one Replica available to process the request", func() {
				it.Before(func() {
					envFake = new(FakeEnvironment)
					request = NewRequestEntity(envFake, subject)

					replicaStock = NewReplicasActiveStock()
					replicaFake = new(fakeReplica)
					replicaStock.Add(replicaFake)

					subject = NewRequestsBufferedStock(envFake, replicaStock, requestsFailedStock)

					subject.Add(request)
				})

				it("schedules the Request to move to a Replica for processing", func() {
					assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[0].From().Name())
					assert.Contains(t, string(envFake.Movements[0].To().Name()), "RequestsProcessing")
				})
			})

			describe("there are no Replicas active", func() {
				describe("scheduling the first retry", func() {
					it.Before(func() {
						envFake = new(FakeEnvironment)
						request = NewRequestEntity(envFake, subject)

						replicaStock = NewReplicasActiveStock()
						subject = NewRequestsBufferedStock(envFake, replicaStock, requestsFailedStock)

						subject.Add(request)
					})

					it("schedules a movement from the Buffer back to itself", func() {
						assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[0].From().Name())
						assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[0].To().Name())
					})

					it("schedules the first retry movement to occur in ~100ms", func() {
						assert.WithinDuration(t, envFake.TheTime.Add(100*time.Millisecond), envFake.Movements[0].OccursAt(), time.Millisecond)
					})
				})

				describe("scheduling subsequent retries", func() {
					it.Before(func() {
						envFake = new(FakeEnvironment)
						request = NewRequestEntity(envFake, subject)

						replicaStock = NewReplicasActiveStock()
						subject = NewRequestsBufferedStock(envFake, replicaStock, requestsFailedStock)

						subject.Add(request)
						subject.Add(request)
					})

					it("on each retry it schedules a movement from Buffer back into itself", func() {
						assert.Equal(t, simulator.MovementKind("buffer_backoff"), envFake.Movements[1].Kind())
						assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[1].From().Name())
						assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[1].To().Name())
					})
				})

				describe("running out of retries", func() {
					it.Before(func() {
						envFake = new(FakeEnvironment)
						request = NewRequestEntity(envFake, subject)

						replicaStock = NewReplicasActiveStock()
						subject = NewRequestsBufferedStock(envFake, replicaStock, requestsFailedStock)

						for i := 0; i < 19; i++ {
							subject.Add(request)
						}
					})

					it("schedules a movement from Buffer into RequestsFailed", func() {
						assert.Equal(t, simulator.MovementKind("exhausted_attempts"), envFake.Movements[18].Kind())
						assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[18].From().Name())
						assert.Equal(t, simulator.StockName("RequestsFailed"), envFake.Movements[18].To().Name())
					})

					it("schedules the movement to occur as soon as possible, offset to prevent collisions", func() {
						assert.Equal(t, envFake.TheTime.Add(1*time.Nanosecond), envFake.Movements[18].OccursAt())
					})
				})
			})
		})
	})
}
