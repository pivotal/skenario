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

	"skenario/pkg/simulator"
)

func TestTrafficSource(t *testing.T) {
	spec.Run(t, "Traffic Source", testTrafficSource, spec.Report(report.Terminal{}))
}

func testTrafficSource(t *testing.T, describe spec.G, it spec.S) {
	var subject TrafficSource
	var rawSubject *trafficSource
	var envFake *FakeEnvironment
	var collectorFake *FakeCollector

	it.Before(func() {
		collectorFake = &FakeCollector{}
		requestsBuffered := NewRequestsBufferedStock(envFake, NewReplicasActiveStock(), simulator.NewSinkStock("RequestsFailed", "Request"), collectorFake)
		envFake = new(FakeEnvironment)

		subject = NewTrafficSource(envFake, requestsBuffered)
		assert.NotNil(t, subject)

		rawSubject = subject.(*trafficSource)
	})

	describe("NewTrafficSource()", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})

		it("sets the buffer", func() {
			assert.Equal(t, simulator.StockName("RequestsBuffered"), rawSubject.requestsBuffered.Name())
		})
	})

	describe("Name()", func() {
		it("is called TrafficSource", func() {
			assert.Equal(t, simulator.StockName("TrafficSource"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Requests", func() {
			assert.Equal(t, simulator.EntityKind("Request"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("gives 0", func() {
			assert.Equal(t, uint64(0), subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("always empty", func() {
			assert.Equal(t, []*simulator.Entity{}, subject.EntitiesInStock())
		})
	})

	describe("Remove()", func() {
		var entity1, entity2 simulator.Entity

		it.Before(func() {
			entity1 = subject.Remove()
			assert.NotNil(t, entity1)
			entity2 = subject.Remove()
			assert.NotNil(t, entity2)
		})

		it("creates a new RequestEntity of EntityKind 'Request'", func() {
			assert.IsType(t, &requestEntity{}, entity1)
			assert.Equal(t, simulator.EntityKind("Request"), entity1.Kind())
		})
	})
}
