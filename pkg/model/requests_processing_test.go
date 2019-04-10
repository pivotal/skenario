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

func TestRequestsProcessing(t *testing.T) {
	spec.Run(t, "RequestsProcessing stock", testRequestsProcessing, spec.Report(report.Terminal{}))
}

func testRequestsProcessing(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestsProcessingStock
	var rawSubject *requestsProcessingStock
	var envFake *fakeEnvironment

	it.Before(func() {
		envFake = new(fakeEnvironment)
		subject = NewRequestsProcessingStock(envFake, 99, simulator.NewSinkStock("RequestsComplete", "Request"))
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
			request = simulator.NewEntity("request-1", simulator.EntityKind("Request"))
			subject.Add(request)
		})

		it("increments the number of requests since last Stat()", func() {
			assert.Equal(t, int32(1), rawSubject.numRequestsSinceLast)
		})

		describe("scheduling processing", func() {
			it("schedules a movement from RequestsProcessing to RequestsComplete", func() {
				assert.Equal(t, simulator.StockName("RequestsComplete"), envFake.movements[0].To().Name())
			})

			it("schedules the movement to occur after 1 second", func() {
				assert.Equal(t, envFake.theTime.Add(1*time.Second), envFake.movements[0].OccursAt())
			})
		})
	})

	describe("RequestCount()", func() {
		it.Before(func() {
			subject.Add(simulator.NewEntity("request-1", "Request"))
			subject.Add(simulator.NewEntity("request-2", "Request"))
		})

		it("Gives the count of requests", func() {
			assert.Equal(t, int32(2), subject.RequestCount())
		})

		it("resets the count each time it is called", func() {
			subject.RequestCount()
			assert.Equal(t, int32(0), subject.RequestCount())
		})
	})
}
