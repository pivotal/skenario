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

func TestUniformRandom(t *testing.T) {
	spec.Run(t, "Uniform random traffic pattern", testUniformRandom, spec.Report(report.Terminal{}))
}

func testUniformRandom(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var config UniformConfig
	var envFake *model.FakeEnvironment
	var collectorFake *model.FakeCollector
	var trafficSource model.TrafficSource
	var bufferStock model.RequestsBufferedStock
	var startAt time.Time
	var runFor time.Duration

	it.Before(func() {
		collectorFake = new(model.FakeCollector)
		envFake = new(model.FakeEnvironment)
		envFake.TheHaltTime = envFake.TheTime.Add(10 * time.Second)
		bufferStock = model.NewRequestsBufferedStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"), collectorFake)
		trafficSource = model.NewTrafficSource(envFake, bufferStock)
		startAt = time.Unix(0, 1)
		runFor = 1 * time.Second

		config = UniformConfig{
			NumberOfRequests: 1000,
			StartAt:          startAt,
			RunFor:           runFor,
		}

		subject = NewUniformRandom(envFake, trafficSource, bufferStock, config)
		subject.Generate()
	})

	describe("Name()", func() {
		it("calls itself 'golang_rand_uniform'", func() {
			assert.Equal(t, "golang_rand_uniform", subject.Name())
		})
	})

	describe("Generate()", func() {
		it("creates 1000 requests", func() {
			assert.Len(t, envFake.Movements, 1000)
		})

		it("created 'arrive_at_buffer' movements", func() {
			for _, mv := range envFake.Movements {
				assert.Equal(t, simulator.MovementKind("arrive_at_buffer"), mv.Kind())
			}
		})

		it("moves from traffic source", func() {
			assert.Equal(t, simulator.StockName("TrafficSource"), envFake.Movements[0].From().Name())
		})

		it("moves to buffer stock", func() {
			assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.Movements[0].To().Name())
		})

		it("created movements between startAt and startAt+runFor", func() {
			for _, mv := range envFake.Movements {
				assert.WithinDuration(t, startAt, mv.OccursAt(), runFor)
			}
		})
	})
}
