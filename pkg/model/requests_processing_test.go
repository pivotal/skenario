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
	"math/rand"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/simulator"
)

func TestRequestsProcessing(t *testing.T) {
	spec.Run(t, "RequestsProcessing stock", testRequestsProcessing, spec.Report(report.Terminal{}))
}

func testRequestsProcessing(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestsProcessingStock
	var rawSubject *requestsProcessingStock
	var envFake *FakeEnvironment

	it.Before(func() {
		envFake = new(FakeEnvironment)
		totalCPUCapacityMillisPerSecond := 100.0
		occupiedCPUCapacityMillisPerSecond := 0.0
		failedSink := simulator.NewSinkStock("RequestsFailed", "Request")
		subject = NewRequestsProcessingStock(envFake, 99, simulator.NewSinkStock("RequestsComplete", "Request"),
			&failedSink, &totalCPUCapacityMillisPerSecond, &occupiedCPUCapacityMillisPerSecond)
		rawSubject = subject.(*requestsProcessingStock)
	})

	describe("NewRequestsProcessingStock()", func() {
		it("sets an Environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})

		it("sets a Requests Completed stock", func() {
			assert.Equal(t, simulator.StockName("RequestsComplete"), rawSubject.requestsComplete.Name())
			assert.Equal(t, simulator.EntityKind("Request"), rawSubject.requestsComplete.KindStocked())
		})

		it("creates a delegate ThroughStock", func() {
			assert.NotNil(t, rawSubject.delegate)
			assert.Equal(t, simulator.StockName("RequestsProcessing"), rawSubject.delegate.Name())
			assert.Equal(t, simulator.EntityKind("Request"), rawSubject.delegate.KindStocked())
		})
	})

	describe("Name()", func() {
		it("includes the replica's name", func() {
			assert.Equal(t, simulator.StockName("RequestsProcessing [99]"), subject.Name())
		})
	})

	describe("Add()", func() {
		var request simulator.Entity

		it.Before(func() {
			bufferStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(), nil)
			request = NewRequestEntity(envFake, bufferStock, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 3 * time.Second})
			subject.Add(request)
		})

		it("increments the number of requests since last Stat()", func() {
			assert.Equal(t, int32(1), rawSubject.numRequestsSinceLast)
		})

		describe("scheduling processing", func() {
			it("schedules a movement from RequestsProcessing to RequestsComplete", func() {
				assert.Equal(t, simulator.StockName("RequestsComplete"), envFake.Movements[0].To().Name())
			})
		})
	})

	describe("RequestCount()", func() {
		it.Before(func() {

			subject.Add(NewRequestEntity(envFake, NewRequestsRoutingStock(envFake, NewReplicasActiveStock(), nil),
				RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
			subject.Add(NewRequestEntity(envFake, NewRequestsRoutingStock(envFake, NewReplicasActiveStock(), nil),
				RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
		})

		it("Gives the count of requests", func() {
			assert.Equal(t, int32(2), subject.RequestCount())
		})

		it("resets the count each time it is called", func() {
			subject.RequestCount()
			assert.Equal(t, int32(0), subject.RequestCount())
		})
	})

	describe("helper functions", func() {
		describe("saturateClamp()", func() {
			describe("when fractionUtilised is greater than 0.96", func() {
				it("returns 0.96", func() {
					assert.Equal(t, saturateClamp(0.97), 0.96)
					assert.Equal(t, saturateClamp(1.01), 0.96)
					assert.Equal(t, saturateClamp(99.0), 0.96)
				})
			})

			describe("when fractionUtilised is between 0.0 and 0.96", func() {
				it("returns fractionUtilised", func() {
					assert.Equal(t, saturateClamp(0.01), 0.01)
					assert.Equal(t, saturateClamp(0.5), 0.5)
					assert.Equal(t, saturateClamp(0.95), 0.95)
				})
			})

			describe("when fractionUtilised is 0", func() {
				it("returns 0.01", func() {
					assert.Equal(t, saturateClamp(0.0), 0.01)
				})
			})

			describe("when fractionUtilised is a negative number", func() {
				it("returns 0.01", func() {
					assert.Equal(t, saturateClamp(-1.0), 0.01)
					assert.Equal(t, saturateClamp(-99.0), 0.01)
				})
			})
		})

		describe("sakasegawaApproximation()", func() {
			it("reduces to the M/M/1 approximation when m = 1", func() {
				assert.Equal(t, 18999999999*time.Nanosecond, sakasegawaApproximation(0.95, 1, time.Second))
			})

			it("approximates 7.3 second slowdown when given 3 replicas and 0.958 utilization", func() {
				assert.Equal(t, 7337661046*time.Nanosecond, sakasegawaApproximation(0.958, 3, time.Second))
			})
		})

		describe("calculateTime()", func() {
			var rng *rand.Rand
			it.Before(func() {
				rng = rand.New(rand.NewSource(1))
			})

			describe("when currentUtilization = 99 %, baseServiceTime = 1 second", func() {
				it("returns base time + random value uniformly selected in range of sakasegawa approximation", func() {
					assert.Equal(t, time.Duration(1068426723), calculateTime(99, time.Second, rng))
				})
			})
		})
	})
}
