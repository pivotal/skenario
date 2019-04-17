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
 *
 */

package trafficpatterns

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/model"
	"skenario/pkg/model/fakes"
	"skenario/pkg/simulator"
)

func TestRamp(t *testing.T) {
	spec.Run(t, "Ramp traffic pattern", testRamp, spec.Report(report.Terminal{}))
}

func testRamp(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var envFake *fakes.FakeEnvironment
	var trafficSource model.TrafficSource
	var bufferStock model.RequestsBufferedStock

	it.Before(func() {
		envFake = new(fakes.FakeEnvironment)
		envFake.TheHaltTime = envFake.TheTime.Add(4*time.Second)
		bufferStock = model.NewRequestsBufferedStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"))
		trafficSource = model.NewTrafficSource(envFake,bufferStock)

		subject = NewRamp(envFake, trafficSource, bufferStock, 1)
		subject.Generate()
	})

	describe("Name()", func() {
		it("calls itself 'ramp'", func() {
			assert.Equal(t, "ramp", subject.Name())
		})
	})

	describe("Generate()", func() {
		it("creates 1 request in the 1st second", func() {
			assert.Equal(t, envFake.TheTime.Add(1*time.Second).Add(1*time.Nanosecond), envFake.Movements[0].OccursAt())
		})

		it("creates 2 requests in the 2nd second", func() {
			assert.Equal(t, envFake.TheTime.Add(2*time.Second).Add(1*time.Nanosecond), envFake.Movements[1].OccursAt())
			assert.Equal(t, envFake.TheTime.Add(2*time.Second).Add(2*time.Nanosecond), envFake.Movements[2].OccursAt())
		})

		it("creates 3 requests in the 3rd second", func() {
			assert.Equal(t, envFake.TheTime.Add(3*time.Second).Add(1*time.Nanosecond), envFake.Movements[3].OccursAt())
			assert.Equal(t, envFake.TheTime.Add(3*time.Second).Add(2*time.Nanosecond), envFake.Movements[4].OccursAt())
			assert.Equal(t, envFake.TheTime.Add(3*time.Second).Add(3*time.Nanosecond), envFake.Movements[5].OccursAt())
		})
	})
}
