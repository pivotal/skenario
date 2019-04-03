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
		bufferStock = NewRequestsBufferedStock(envFake, NewReplicasActiveStock(), nil)
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

	describe("NextBackoff()", func() {
		var backoff time.Duration
		var outOfAttempts bool

		describe("scheduling the first retry", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				backoff, outOfAttempts = subject.NextBackoff()
			})

			it("returns the next backoff duration", func() {
				assert.Equal(t, 100*time.Millisecond, backoff)
			})

			// yes, this test can pass by coincidence, but at least it documents the intent
			it("returns false to indicate that more retries are possible", func() {
				assert.False(t, outOfAttempts)
			})
		})

		describe("scheduling subsequent retries", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				backoff, _ = subject.NextBackoff()
			})

			// The real implementation has jitter, but I will ignore this for now.
			it("on each retry, increases the backoff time by 1.3x", func() {
				backoff, _ = subject.NextBackoff()
				assert.Equal(t, 130*time.Millisecond, backoff)
				backoff, _ = subject.NextBackoff()
				assert.Equal(t, 169*time.Millisecond, backoff)
			})
		})

		describe("running out of retries", func() {
			it.Before(func() {
				subject = NewRequestEntity(envFake, bufferStock)
				for i := 0; i < 18; i++ {
					_, outOfAttempts = subject.NextBackoff()
				}
				assert.False(t, outOfAttempts)
			})

			it("returns true to indicate no more retries are possible", func() {
				_, outOfAttempts = subject.NextBackoff()
				assert.True(t, outOfAttempts)
			})
		})
	})
}
