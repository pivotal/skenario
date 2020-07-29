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
	"math"
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
		envFake = NewFakeEnvironment()
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
		var request RequestEntity

		describe("there is no free cpu resource", func() {
			it.Before(func() {
				*rawSubject.occupiedCPUCapacityMillisPerSecond = *rawSubject.totalCPUCapacityMillisPerSecond
				routingStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil)
				request = NewRequestEntity(envFake, routingStock, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 3 * time.Second})
				subject.Add(request)
			})
			describe("scheduling processing", func() {
				it("schedules a movement from RequestsProcessing to RequestsFailed", func() {
					assert.Equal(t, simulator.StockName("RequestsFailed"), envFake.Movements[0].To().Name())
				})
			})

			it("there is 1 entity in processingStock", func() {
				assert.Equal(t, uint64(1), subject.Count())
			})
		})
		describe("request fails as request total time exceeds request timeout", func() {
			it.Before(func() {
				routingStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil)
				request = NewRequestEntity(envFake, routingStock, RequestConfig{CPUTimeMillis: 20000, IOTimeMillis: 200, Timeout: 3 * time.Second})
				subject.Add(request)
			})

			it("cpu resource is allocated", func() {
				assert.NotEqual(t, rawSubject.occupiedCPUCapacityMillisPerSecond, 0)
			})
			it("schedules a movement from RequestsProcessing to RequestsFailed", func() {
				assert.Equal(t, simulator.StockName("RequestsFailed"), envFake.Movements[0].To().Name())
			})

			describe("when request timeout is over", func() {
				it("time for movement from RequestsProcessing to RequestsFailed = current time + timeout ", func() {
					assert.Equal(t, envFake.TheTime.Add(request.(*requestEntity).requestConfig.Timeout), envFake.Movements[0].OccursAt())
				})
			})
		})

		describe("request completes as request total time doesn't exceed request timeout", func() {
			it.Before(func() {
				//there is free cpu resource
				*rawSubject.occupiedCPUCapacityMillisPerSecond = 0.0
				routingStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil)
				request = NewRequestEntity(envFake, routingStock, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 3 * time.Second})
				subject.Add(request)
			})

			it("90.909 cpu resource is allocated", func() {
				assert.Less(t, math.Abs(*rawSubject.occupiedCPUCapacityMillisPerSecond-90.909), 0.001)
			})

			it("schedules a movement from RequestsProcessing to RequestsComplete", func() {
				assert.Equal(t, simulator.StockName("RequestsComplete"), envFake.Movements[0].To().Name())
			})
			it("allocated cpu resource for a request is freed", func() {
				rawSubject.Remove(nil)
				assert.Less(t, math.Abs(*rawSubject.occupiedCPUCapacityMillisPerSecond-0.0), 0.001)
			})
		})
	})

	describe("RequestCount()", func() {
		it.Before(func() {

			subject.Add(NewRequestEntity(envFake, NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil),
				RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 1 * time.Second}))
			subject.Add(NewRequestEntity(envFake, NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil),
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

	describe("computation of cpu utilization", func() {
		describe("when totalCPUCapacityMillisPerSecond = 100.0 occupiedCPUCapacityMillisPerSecond = 60.0", func() {

			it.Before(func() {
				*rawSubject.totalCPUCapacityMillisPerSecond = 100.0
				*rawSubject.occupiedCPUCapacityMillisPerSecond = 60.0
			})
			describe("request: CPUTimeMillis = 200, IOTimeMillis = 200, Timeout = 3 sec", func() {
				var request requestEntity
				var isRequestSuccessful bool
				var totalTime time.Duration
				it.Before(func() {
					routingStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil)
					request = *NewRequestEntity(envFake, routingStock, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 3 * time.Second}).(*requestEntity)
					rawSubject.calculateCPUUtilizationForRequest(request, &totalTime, &isRequestSuccessful)
				})
				it("request total time ~ 5.6s (5.2s + delay time)", func() {
					assert.Less(t, math.Abs(float64(totalTime)/float64(time.Second)-5.6), 0.4)
				})
				it("request is not successful, because request timeout 3s, but total time is more", func() {
					assert.False(t, isRequestSuccessful)
				})

			})

		})

		describe("when totalCPUCapacityMillisPerSecond = 100.0 occupiedCPUCapacityMillisPerSecond = 0.0", func() {
			it.Before(func() {
				*rawSubject.totalCPUCapacityMillisPerSecond = 100.0
				*rawSubject.occupiedCPUCapacityMillisPerSecond = 0.0
			})
			describe("request: CPUTimeMillis = 200, IOTimeMillis = 200, Timeout = 3 sec", func() {
				var request requestEntity
				var isRequestSuccessful bool
				var totalTime time.Duration
				it.Before(func() {
					routingStock := NewRequestsRoutingStock(envFake, NewReplicasActiveStock(envFake), nil)
					request = *NewRequestEntity(envFake, routingStock, RequestConfig{CPUTimeMillis: 200, IOTimeMillis: 200, Timeout: 3 * time.Second}).(*requestEntity)
					rawSubject.calculateCPUUtilizationForRequest(request, &totalTime, &isRequestSuccessful)
				})
				it("request total time ~ 2.25s (2.2s + delay time)", func() {
					assert.Less(t, math.Abs(float64(totalTime)/float64(time.Second)-2.25), 0.05)
				})
				it("request is successful, because request timeout 3s, but total time is less ", func() {
					assert.True(t, isRequestSuccessful)
				})

			})

		})
	})
}
