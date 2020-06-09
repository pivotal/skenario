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

package trafficpatterns

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

func TestStep(t *testing.T) {
	spec.Run(t, "Ramp traffic pattern", testStep, spec.Report(report.Terminal{}))
}

func testStep(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var config StepConfig
	var envFake *model.FakeEnvironment
	var trafficSource model.TrafficSource
	var routingStock model.RequestsRoutingStock

	it.Before(func() {
		envFake = new(model.FakeEnvironment)
		envFake.TheHaltTime = envFake.TheTime.Add(20 * time.Second)
		routingStock = model.NewRequestsRoutingStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"))
		trafficSource = model.NewTrafficSource(envFake, routingStock)

		config = StepConfig{
			RPS:       10,
			StepAfter: 10 * time.Second,
		}
		subject = NewStep(envFake, trafficSource, routingStock, config)
	})

	describe("Name()", func() {
		it("calls itself 'Step'", func() {
			assert.Equal(t, "step", subject.Name())
		})
	})

	describe("Generate()", func() {
		it.Before(func() {
			subject.Generate()
		})

		describe("constant RPS", func() {
			it("schedules 10 requests in the first step second", func() {
				for i := 0; i < 10; i++ {
					assert.WithinDuration(t, envFake.TheTime.Add(10500*time.Millisecond), envFake.Movements[i].OccursAt(), 500*time.Millisecond)
				}
			})

			it("schedules 10 requests in the last second of the simulation", func() {
				for i := 90; i < 100; i++ {
					assert.WithinDuration(t, envFake.TheTime.Add(19500*time.Millisecond), envFake.Movements[i].OccursAt(), 500*time.Millisecond)
				}
			})
		})

		describe("stepAfter time", func() {
			var startAt time.Time

			it.Before(func() {
				startAt = envFake.TheTime.Add(10 * time.Second)
			})

			it("does not schedule any requests before stepAfter", func() {
				for _, mv := range envFake.Movements {
					assert.True(t, mv.OccursAt().After(startAt))
				}
			})
		})

		describe("the total number of requests", func() {
			it("generates rps * (haltTime - stepAfter) requests in total", func() {
				assert.Len(t, envFake.Movements, 100)
			})
		})

	})
}
