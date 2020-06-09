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

func TestSinusoidal(t *testing.T) {
	spec.Run(t, "Sinusoidal traffic pattern", testSinusoidal, spec.Report(report.Terminal{}))
}

func testSinusoidal(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var config SinusoidalConfig
	var envFake *model.FakeEnvironment
	var amplitude int
	var period time.Duration
	var trafficSource model.TrafficSource
	var routingStock model.RequestsRoutingStock

	it.Before(func() {
		amplitude = 20
		period = 20 * time.Second

		envFake = new(model.FakeEnvironment)
		envFake.TheTime = time.Unix(0, 0)
		envFake.TheHaltTime = envFake.TheTime.Add(30 * time.Second)

		routingStock = model.NewRequestsRoutingStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"))
		trafficSource = model.NewTrafficSource(envFake, routingStock)
		config = SinusoidalConfig{
			Amplitude: amplitude,
			Period:    period,
		}
		subject = NewSinusoidal(envFake, trafficSource, routingStock, config)
	})

	describe("Name()", func() {
		it("calls itself 'sinusoidal'", func() {
			assert.Equal(t, "sinusoidal", subject.Name())
		})
	})

	describe("Generate()", func() {
		var expectedRPS = []int{
			20, 26, 32, 36, 39, 40, 39, 36, 32, 26, 20, 14, 8, 4, 1,
			0, 1, 4, 8, 14, 20, 26, 32, 36, 39, 40, 39, 36, 32, 26,
		}

		it.Before(func() {
			subject.Generate()
		})

		it("produces 726 requests in total", func() {
			assert.Len(t, envFake.Movements, 726)
		})

		it("produces a sinusoidal pattern", func() {
			mvmtIdx := 0
			sec := time.Unix(1, 0)

			for _, v := range expectedRPS {
				for i := 0; i < v; i++ {
					assert.WithinDuration(t, sec, envFake.Movements[mvmtIdx].OccursAt(), time.Second)
					mvmtIdx++
				}
				sec = sec.Add(time.Second)
			}
		})
	})
}
