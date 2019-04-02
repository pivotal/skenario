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

	"knative-simulator/pkg/simulator"
)

func TestRequestEntity(t *testing.T) {
	spec.Run(t, "Request entities", testRequestEntity, spec.Report(report.Terminal{}))
}

func testRequestEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject RequestEntity
	var rawSubject *requestEntity
	var envFake *fakeEnvironment
	var bufferStock RequestsBufferedStock

	it.Before(func() {
		bufferStock = NewRequestsBufferedStock(envFake, NewReplicasActiveStock())
		envFake = new(fakeEnvironment)
		subject = NewRequestEntity(envFake, bufferStock)
		rawSubject = subject.(*requestEntity)
	})

	describe("NewRequestEntity()", func() {
		it("sets an Environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})

		it("sets the buffer stock", func() {
			assert.Equal(t, bufferStock, rawSubject.bufferStock)
		})

		it("sets the retry backoff duration to 100ms", func() {
			assert.Equal(t, 100*time.Millisecond, rawSubject.nextBackoff)
		})
	})

	describe("Entity interface", func() {
		it("implements Name()", func() {
			assert.Equal(t, simulator.EntityName("request-1"), subject.Name())
		})

		it("gives sequential Name()s", func() {
			subject2 := NewRequestEntity(envFake, bufferStock)
			assert.Equal(t, simulator.EntityName("request-2"), subject2.Name())
		})

		it("implements Kind()", func() {
			assert.Equal(t, simulator.EntityKind("Request"), subject.Kind())
		})
	})

	describe("ScheduleBackoffMovement()", func() {
		var outOfAttempts bool

		describe("scheduling the first retry", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				outOfAttempts = subject.ScheduleBackoffMovement()
			})
			it("schedules a movement from the Buffer back to itself", func() {
				assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.movements[0].From().Name())
				assert.Equal(t, simulator.StockName("RequestsBuffered"), envFake.movements[0].To().Name())
			})

			it("gives sequential MovementKinds", func() {
				assert.Equal(t, simulator.MovementKind("buffer_backoff_1"), envFake.movements[0].Kind())
				subject.ScheduleBackoffMovement()
				assert.Equal(t, simulator.MovementKind("buffer_backoff_2"), envFake.movements[1].Kind())
			})

			it("schedules the movement to occur in 100ms", func() {
				assert.Equal(t, envFake.theTime.Add(100*time.Millisecond), envFake.movements[0].OccursAt())
			})

			// yes, this test can pass by coincidence, but at least it documents the intent
			it("returns false to indicate that more retries are possible", func() {
				assert.False(t, outOfAttempts)
			})
		})

		describe("scheduling subsequent retries", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				rawSubject = subject.(*requestEntity)
				subject.ScheduleBackoffMovement()
			})

			// The real implementation has jitter, but I will ignore this for now.
			it("on each retry, increases the backoff time by 1.3x", func() {
				assert.Equal(t, 130*time.Millisecond, rawSubject.nextBackoff)
				subject.ScheduleBackoffMovement()
				assert.Equal(t, 169*time.Millisecond, rawSubject.nextBackoff)
			})
		})

		describe("running out of retries", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				rawSubject = subject.(*requestEntity)
				for i := 0; i < 18; i++ {
					outOfAttempts = subject.ScheduleBackoffMovement()
				}
				assert.False(t, outOfAttempts)
			})

			it("returns true to indicate no more retries are possible", func() {
				outOfAttempts = subject.ScheduleBackoffMovement()
				assert.True(t, outOfAttempts)
			})
		})
	})
}
