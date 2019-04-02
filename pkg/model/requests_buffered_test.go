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

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/simulator"
)

func TestRequestsBuffered(t *testing.T) {
	spec.Run(t, "RequestsBuffered stock", testRequestsBuffered, spec.Report(report.Terminal{}))
}

func testRequestsBuffered(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestsBufferedStock
	var rawSubject *requestsBufferedStock
	var envFake *fakeEnvironment
	var replicaStock ReplicasActiveStock
	var replicaFake *fakeReplica

	describe("NewRequestsBufferedStock()", func() {
		it.Before(func() {
			envFake = new(fakeEnvironment)
			replicaStock = NewReplicasActiveStock()
			subject = NewRequestsBufferedStock(envFake, replicaStock)
			rawSubject = subject.(*requestsBufferedStock)
		})
		it("creates a delegate ThroughStock", func() {
			assert.NotNil(t, rawSubject.delegate)
			assert.Equal(t, simulator.StockName("RequestsBuffered"), rawSubject.delegate.Name())
			assert.Equal(t, simulator.EntityKind("Request"), rawSubject.delegate.KindStocked())
		})
	})

	describe("Add()", func() {
		describe("there are no other requests yet", func() {
			describe("there is at least one Replica available to process the request", func() {
				it.Before(func() {
					envFake = new(fakeEnvironment)

					replicaStock = NewReplicasActiveStock()
					replicaFake = new(fakeReplica)
					replicaStock.Add(replicaFake)

					subject = NewRequestsBufferedStock(envFake, replicaStock)

					subject.Add(simulator.NewEntity("request-111", "Request"))
				})

				it("schedules the Request to move to a Replica for processing", func() {
					assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.movements[0].From().Name())
					assert.Contains(t, string(envFake.movements[0].To().Name()), "RequestsProcessing")
				})
			})

			describe("there are no Replicas active", func() {
				it.Before(func() {
					envFake = new(fakeEnvironment)
					replicaStock = NewReplicasActiveStock()

					subject = NewRequestsBufferedStock(envFake, replicaStock)

					subject.Add(simulator.NewEntity("request-999", "Request"))
				})

				it.Pend("schedules a backoff using a delay given by Request Entity", func() {

					// assert that RequestEntity.ScheduleBackoffMovement was called
				})

				describe.Pend("the Request says it has run out of retries", func() {
					it("RequestsBuffered schedules a movement into the RequestsFailed sink", func() {

					})
				})
			})
		})

		describe.Pend("there are other requests waiting in the buffer", func() {
			it.Pend("assigns the Requests to Replicas in a round robin", func() {
				// TODO is this the actual behaviour? If not, does it matter?
			})
		})

	})
}
