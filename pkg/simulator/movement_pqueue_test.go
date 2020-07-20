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

package simulator

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestMovementPQ(t *testing.T) {
	spec.Run(t, "Movement priority queue", testMovementPQ, spec.Report(report.Terminal{}))
}

func testMovementPQ(t *testing.T, describe spec.G, it spec.S) {
	var subject MovementPriorityQueue
	var movement Movement
	var theTime, scheduledAt time.Time
	var shifted bool
	var err error

	describe("EnqueueMovement()", func() {
		it.Before(func() {
			theTime = time.Now()
			movement = NewMovement("test movement kind", theTime, nil, nil, nil)
		})

		describe("when there is an existing Movement scheduled at the same time", func() {
			it.Before(func() {
				subject = NewMovementPriorityQueue()

				_, _, err = subject.EnqueueMovement(movement)
				assert.NoError(t, err)

				shifted, scheduledAt, err = subject.EnqueueMovement(movement)
				assert.NoError(t, err)

			})
			it("time-shifts the Movement to the next free time", func() {
				assert.Equal(t, theTime.Add(1*time.Nanosecond), scheduledAt)
			})

			it("indicates that time-shifting occurred", func() {
				assert.True(t, shifted)
			})
		})

		describe("when no other Movement has been scheduled at the same time", func() {
			it.Before(func() {
				subject = NewMovementPriorityQueue()

				shifted, scheduledAt, err = subject.EnqueueMovement(movement)
				assert.NoError(t, err)
			})

			it("does not time-shift the Movement", func() {
				assert.Equal(t, theTime, scheduledAt)
			})

			it("indicates that time-shifting did not occur", func() {
				assert.False(t, shifted)
			})
		})
	})

	describe("DequeueMovement()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, nil)
		})

		it("returns Movements", func() {
			var dqmv Movement
			var err error
			_, _, err = subject.EnqueueMovement(movement)
			assert.NoError(t, err)

			dqmv, err, _ = subject.DequeueMovement()

			subject.Close()

			assert.NoError(t, err)
			assert.Equal(t, movement, dqmv)
		})

		it("returns a 'closed' flag to indicate whether the queue has closed", func() {
			var closed bool
			var err error

			subject.Close()
			mv, err, closed := subject.DequeueMovement()

			assert.Nil(t, mv)
			assert.NoError(t, err)
			assert.True(t, closed)

		})
	})

	describe("Close()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, nil)
		})

		it("closes the heap", func() {
			subject.Close()
			assert.True(t, subject.IsClosed())
		})
	})

	describe("IsClosed()", func() {
		it.Before(func() {
			subject = NewMovementPriorityQueue()
			movement = NewMovement("test movement kind", time.Now(), nil, nil, nil)
		})

		it("starts false", func() {
			assert.False(t, subject.IsClosed())
		})
	})

	describe("helpers", func() {
		describe("movementToKey()", func() {
			it.Before(func() {
				movement = NewMovement("test movement kind", time.Unix(0, 111000111), nil, nil, nil)
			})

			it("returns the OccursAt() as a string", func() {
				key, err := movementToKey(movement)
				assert.NoError(t, err)
				assert.Equal(t, "111000111", key)
			})
		})

		describe("leftMovementIsEarlier()", func() {
			var earlier, later Movement

			it.Before(func() {
				earlier = NewMovement("test movement kind", time.Unix(111, 0), nil, nil, nil)
				later = NewMovement("test movement kind", time.Unix(999, 0), nil, nil, nil)
			})

			describe("when the first argument is earlier", func() {
				it("returns true", func() {
					assert.True(t, leftMovementIsEarlier(earlier, later))
				})
			})

			describe("when the second argument is earlier", func() {
				it("returns false", func() {
					assert.False(t, leftMovementIsEarlier(later, earlier))
				})
			})
		})
	})
}
