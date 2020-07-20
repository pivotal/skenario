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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestEnvironment(t *testing.T) {
	spec.Run(t, "Environment spec", testEnvironment, spec.Report(report.Terminal{}))
}

func testEnvironment(t *testing.T, describe spec.G, it spec.S) {
	var (
		subject   Environment
		ctx       context.Context
		movement  Movement
		fromStock SourceStock
		toStock   SinkStock
		startTime time.Time
		runFor    time.Duration
	)

	it.Before(func() {
		ctx = context.Background()
		startTime = time.Unix(222222, 0)
		runFor = 555555 * time.Second
		fromStock = &EchoSourceStockType{
			name: "from stock",
			kind: "test entity kind",
		}
		toStock = NewSinkStock("to stock", "test entity kind")
	})

	describe("NewEnvironment()", func() {
		var completed []CompletedMovement
		var ignored []IgnoredMovement
		var err error
		completedNotes := make([]string, 0)
		ignoredNotes := make([]string, 0)

		it.Before(func() {
			subject = NewEnvironment(ctx, startTime, runFor)
			assert.NotNil(t, subject)

			completed, ignored, err = subject.Run()
			assert.NoError(t, err)

			for _, c := range completed {
				for _, n := range c.Movement.Notes() {
					completedNotes = append(completedNotes, n)
				}
			}

			for _, i := range ignored {
				for _, n := range i.Movement.Notes() {
					ignoredNotes = append(ignoredNotes, n)
					fmt.Println(i.Reason)
				}
			}
		})

		it("adds a start scenario movement", func() {
			assert.Contains(t, completedNotes, "Start scenario")
		})

		it("adds a halt scenario movement", func() {
			assert.Contains(t, completedNotes, "Halt scenario")
		})

		it("doesn't ignore the start or halt scenario movements", func() {
			assert.NotContains(t, ignoredNotes, "Start scenario")
			assert.NotContains(t, ignoredNotes, "Halt scenario")
		})
	}, spec.Nested())

	describe("AddToSchedule()", func() {
		it.Before(func() {
			subject = NewEnvironment(ctx, startTime, runFor)
			assert.NotNil(t, subject)
		})

		describe("the scheduled movement will occur during the simulation", func() {
			it("returns true", func() {
				movement = NewMovement("test movement kind", time.Unix(333333, 0), fromStock, toStock)
				assert.True(t, subject.AddToSchedule(movement))
			})
		})

		describe("the scheduled movement would occur at halt", func() {
			it("returns false", func() {
				movement = NewMovement("test movement kind", time.Unix(777777, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the scheduled movement would occur after the simulation halts", func() {
			it("returns false", func() {
				movement = NewMovement("test movement kind", time.Unix(999999, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the movement would occur before the current simulation time", func() {
			it("returns false", func() {
				movement = NewMovement("test movement kind", time.Unix(111111, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the movement would occur at the current simulation time", func() {
			it("returns false", func() {
				movement = NewMovement("test movement kind", time.Unix(222222, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})
	}, spec.Nested())

	describe("Run()", func() {
		describe("taking the next movement from the schedule", func() {
			var fromMock, toMock *MockStockType
			var e Entity
			var err error

			it.Before(func() {
				subject = NewEnvironment(ctx, startTime, runFor)
				assert.NotNil(t, subject)

				fromMock = new(MockStockType)
				toMock = new(MockStockType)
				e = NewEntity("test entity", "mock kind")
				fromMock.On("Remove").Return(e)
				toMock.On("Add", e).Return(nil)

				movement = NewMovement("test movement kind", time.Unix(333333, 0), fromMock, toMock)

				subject.AddToSchedule(movement)
				_, _, err = subject.Run()
				assert.NoError(t, err)
			})

			it("Remove()s from the 'from' stock", func() {
				fromMock.AssertCalled(t, "Remove")
			})

			it("Add()s to the 'to' stock", func() {
				toMock.AssertCalled(t, "Add", e)
			})
		})

		describe("results", func() {
			describe("completed movements", func() {
				var first, second Movement
				var completed []CompletedMovement

				it.Before(func() {
					var err error

					subject = NewEnvironment(ctx, startTime, runFor)
					assert.NotNil(t, subject)

					first = NewMovement("test movement kind", time.Unix(333333, 0), fromStock, toStock)
					second = NewMovement("test movement kind", time.Unix(444444, 0), fromStock, toStock)

					subject.AddToSchedule(first)
					subject.AddToSchedule(second)
					completed, _, err = subject.Run()

					assert.NoError(t, err)
				})

				it("contains the correct number of completed movements", func() {
					assert.Len(t, completed, 4) // start scenario, halt scenario, first, second
				})

				it("contains the completed movements", func() {
					assert.Equal(t, first, completed[1].Movement)
					assert.Equal(t, second, completed[2].Movement)
				})

				it("contains the moved entities", func() {
					assert.Equal(t, EntityName("entity-1"), completed[2].Moved.Name())
				})
			})

			describe("ignored movements", func() {
				var nilStock ThroughStock
				var tooEarly, tooLate, goldilocks, collides, nilEntity Movement
				var ignored []IgnoredMovement

				it.Before(func() {
					subject = NewEnvironment(ctx, startTime, runFor)
					assert.NotNil(t, subject)

					nilStock = NewThroughStock("NilStock", "test movement kind")

					var err error

					tooEarly = NewMovement("test movement kind", time.Unix(111111, 0), fromStock, toStock)
					goldilocks = NewMovement("test movement kind", time.Unix(333333, 0), fromStock, toStock)
					collides = NewMovement("test movement kind", time.Unix(333333, 0), fromStock, toStock)
					tooLate = NewMovement("test movement kind", time.Unix(999999, 0), fromStock, toStock)
					nilEntity = NewMovement("test movement kind", time.Unix(444444, 0), nilStock, toStock)

					subject.AddToSchedule(tooEarly)
					subject.AddToSchedule(goldilocks)
					subject.AddToSchedule(collides)
					subject.AddToSchedule(tooLate)
					subject.AddToSchedule(nilEntity)
					_, ignored, err = subject.Run()

					assert.NoError(t, err)
				})

				it("contains the correct number of ignored movements", func() {
					assert.Len(t, ignored, 3)
				})

				it("contains movements that were scheduled in the past", func() {
					assert.Contains(t, ignored, IgnoredMovement{Reason: OccursInPast, Movement: tooEarly})
				})

				it("contains movements that were scheduled after the halt", func() {
					assert.Contains(t, ignored, IgnoredMovement{Reason: OccursAfterHalt, Movement: tooLate})
				})

				it("contains movements for which the entity was nil", func() {
					assert.Contains(t, ignored, IgnoredMovement{Reason: FromStockIsEmpty, Movement: nilEntity})
				})

				it("doesn't contain any events that were scheduled", func() {
					assert.NotContains(t, ignored, IgnoredMovement{Reason: OccursInPast, Movement: goldilocks})
					assert.NotContains(t, ignored, IgnoredMovement{Reason: OccursAfterHalt, Movement: goldilocks})
				})
			})
		})
	}, spec.Nested())

	describe("CurrentMovementTime()", func() {
		it.Before(func() {
			subject = NewEnvironment(ctx, startTime, runFor)
			assert.NotNil(t, subject)
		})

		it("gives the time of the movement currently in progress", func() {
			assert.Equal(t, startTime, subject.CurrentMovementTime())
			subject.Run()
			assert.Equal(t, startTime.Add(runFor), subject.CurrentMovementTime())
		})
	})

	describe("HaltTime()", func() {
		it.Before(func() {
			subject = NewEnvironment(ctx, startTime, runFor)
			assert.NotNil(t, subject)
		})

		it("gives the start time + run duration", func() {
			assert.Equal(t, startTime.Add(runFor), subject.HaltTime())
		})
	})

	describe("Context()", func() {
		it.Before(func() {
			subject = NewEnvironment(ctx, startTime, runFor)
			assert.NotNil(t, subject)
		})

		it("returns the context given at creation", func() {
			assert.Equal(t, ctx, subject.Context())
		})
	})

	describe("helper funcs", func() {
		describe("newEnvironment()", func() {
			var rawSubject *environment
			var mpq MovementPriorityQueue

			it.Before(func() {
				mpq = NewMovementPriorityQueue()
				rawSubject = newEnvironment(ctx, time.Unix(0, 0), time.Minute, mpq)
			})

			it("configures the halted scenario stock to use haltingStock", func() {
				assert.Equal(t, rawSubject.haltedScenario.Name(), StockName("HaltedScenario"))
				assert.IsType(t, &haltingSink{}, rawSubject.haltedScenario)
			})

			it("sets a context", func() {
				assert.Equal(t, ctx, rawSubject.ctx)
			})
		})
	}, spec.Nested())
}
