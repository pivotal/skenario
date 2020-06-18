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
	"testing"
	"time"

	"skenario/pkg/simulator"
)

func TestRequestsRouting(t *testing.T) {
	spec.Run(t, "RequestsRouting stock", testRequestsRouting, spec.Report(report.Terminal{}))
}

func testRequestsRouting(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestsRoutingStock
	var rawSubject *requestsRoutingStock
	var envFake *FakeEnvironment
	var replicaStock ReplicasActiveStock
	var requestsFailedStock simulator.SinkStock
	var replicaFake *FakeReplica

	it.Before(func() {
		requestsFailedStock = simulator.NewSinkStock("RequestsFailed", "Request")
	})

	describe("NewRoutingStock()", func() {
		it.Before(func() {
			envFake = new(FakeEnvironment)
			replicaStock = NewReplicasActiveStock()
			subject = NewRequestsRoutingStock(envFake, replicaStock, nil)
			rawSubject = subject.(*requestsRoutingStock)
		})

		it("creates a delegate ThroughStock", func() {
			assert.NotNil(t, rawSubject.delegate)
			assert.Equal(t, simulator.StockName("RequestsRouting"), rawSubject.delegate.Name())
			assert.Equal(t, simulator.EntityKind("Request"), rawSubject.delegate.KindStocked())
		})
	})

	describe("Add()", func() {
		var request RequestEntity

		describe("there are multiple replicas available to serve multiple requests", func() {
			it.Before(func() {
				envFake = new(FakeEnvironment)

				replicaStock = NewReplicasActiveStock()

				replicaFake = new(FakeReplica)
				replicaFake.FakeReplicaNum = 11
				replicaStock.Add(replicaFake)

				replicaFake = new(FakeReplica)
				replicaFake.FakeReplicaNum = 22
				replicaStock.Add(replicaFake)

				replicaFake = new(FakeReplica)
				replicaFake.FakeReplicaNum = 33
				replicaStock.Add(replicaFake)

				subject = NewRequestsRoutingStock(envFake, replicaStock, requestsFailedStock)

				subject.Add(NewRequestEntity(envFake, subject, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
				subject.Add(NewRequestEntity(envFake, subject, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
				subject.Add(NewRequestEntity(envFake, subject, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
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
					request = NewRequestEntity(envFake, subject, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second})

					replicaStock = NewReplicasActiveStock()
					replicaFake = new(FakeReplica)
					replicaStock.Add(replicaFake)

					subject = NewRequestsRoutingStock(envFake, replicaStock, requestsFailedStock)

					subject.Add(request)
				})

				it("schedules the Request to move to a Replica for processing", func() {
					assert.Equal(t, simulator.StockName("RequestsRouting"), envFake.Movements[0].From().Name())
					assert.Contains(t, string(envFake.Movements[0].To().Name()), "RequestsProcessing")
				})
			})
		})
	})
}
