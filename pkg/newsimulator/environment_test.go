package newsimulator

import (
	"fmt"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockStockType struct {
	mock.Mock
}

func (mss *mockStockType) Name() StockName {
	mss.Called()
	return StockName("mock source")
}

func (mss *mockStockType) KindStocked() EntityKind {
	mss.Called()
	return EntityKind("mock kind")
}
func (mss *mockStockType) Count() uint64 {
	mss.Called()
	return uint64(0)
}
func (mss *mockStockType) Remove() Entity {
	mss.Called()
	return NewEntity("test entity", "mock kind")
}
func (mss *mockStockType) Add(entity Entity) error {
	mss.Called(entity)
	return nil
}

func TestEnvironment(t *testing.T) {
	spec.Run(t, "Environment spec", testEnvironment, spec.Report(report.Terminal{}))
}

// We hand-roll the echo source stock, otherwise the compiler will use ThroughStock,
// leading to nil errors when we try to .Remove() a non-existent entry.
type echoSourceStockType struct {
	name StockName
	kind EntityKind
	series int
}

func (es *echoSourceStockType) Name() StockName {
	return es.name
}

func (es *echoSourceStockType) KindStocked() EntityKind {
	return es.kind
}

func (es *echoSourceStockType) Count() uint64 {
	return 0
}

func (es *echoSourceStockType) Remove() Entity {
	name := EntityName(fmt.Sprintf("entity-%d", es.series))
	es.series++
	return NewEntity(name, es.kind)
}

func testEnvironment(t *testing.T, describe spec.G, it spec.S) {
	var (
		subject   Environment
		movement  Movement
		fromStock SourceStock
		toStock   SinkStock
		startTime time.Time
	)

	startTime = time.Unix(222222, 0)

	it.Before(func() {
		subject = NewEnvironment(startTime, 555555*time.Second)
		assert.NotNil(t, subject)

		fromStock = &echoSourceStockType{
			name:   "from stock",
			kind:   "test entity kind",
		}
		toStock = NewSinkStock("to stock", "test entity kind")
	})

	describe("AddToSchedule()", func() {
		describe("the scheduled movement will occur during the simulation", func() {
			it("returns true", func() {
				movement = NewMovement(time.Unix(333333, 0), fromStock, toStock)
				assert.True(t, subject.AddToSchedule(movement))
			})
		})

		describe("the scheduled movement would occur after the simulation ends", func() {
			it("returns false", func() {
				movement = NewMovement(time.Unix(999999, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the movement would occur at or before the current simulation time", func() {
			it("returns false", func() {
				movement = NewMovement(time.Unix(222222, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))

				movement = NewMovement(time.Unix(111111, 0), fromStock, toStock)
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe.Pend("Schedule listeners", func() {
			it("calls OnSchedule() on registered listeners", func() {

			})
		})
	})

	describe("Run()", func() {
		describe("taking the next movement from the schedule", func() {
			var fromMock, toMock *mockStockType
			var e Entity

			it.Before(func() {
				fromMock = new(mockStockType)
				toMock = new(mockStockType)
				e = NewEntity("test entity", "mock kind")
				fromMock.On("Remove").Return(e)
				toMock.On("Add", e).Return(nil)

				movement = NewMovement(time.Unix(333333, 0), fromMock, toMock)

				subject.AddToSchedule(movement)
				_, _, err := subject.Run()
				assert.NoError(t, err)
			})

			it("Remove()s from the 'from' stock", func() {
				fromMock.AssertCalled(t, "Remove")
			})

			it("Add()s to the 'to' stock", func() {
				toMock.AssertCalled(t, "Add", e)
			})
		})

		describe.Pend("Start-simulation movement", func() {

		})

		describe.Pend("Halt-simulation movement", func() {

		})

		describe("results", func() {
			describe("completed movements", func() {
				var first, second Movement
				var completed []CompletedMovement

				it.Before(func() {
					var err error

					subject = NewEnvironment(startTime, 555555*time.Second)

					first = NewMovement(time.Unix(333333, 0), fromStock, toStock)
					second = NewMovement(time.Unix(444444, 0), fromStock, toStock)

					subject.AddToSchedule(first)
					subject.AddToSchedule(second)
					completed, _, err = subject.Run()

					assert.NoError(t, err)
				})

				it("contains the correct number of completed movements", func() {
					assert.Len(t, completed, 2)
				})

				it("contains the completed movements", func() {
					assert.Contains(t, completed, CompletedMovement{movement: first})
					assert.Contains(t, completed, CompletedMovement{movement: second})
				})
			})

			describe("ignored movements", func() {
				var tooEarly, tooLate, goldilocks Movement
				var ignored []IgnoredMovement

				it.Before(func() {
					var err error

					tooEarly = NewMovement(time.Unix(111111, 0), fromStock, toStock)
					goldilocks = NewMovement(time.Unix(333333, 0), fromStock, toStock)
					tooLate = NewMovement(time.Unix(999999, 0), fromStock, toStock)

					subject.AddToSchedule(tooEarly)
					subject.AddToSchedule(goldilocks)
					subject.AddToSchedule(tooLate)
					_, ignored, err = subject.Run()

					assert.NoError(t, err)
				})

				it("contains the correct number of ignored movements", func() {
					assert.Len(t, ignored, 2)
				})

				it("contains movements that were scheduled in the past", func() {
					assert.Contains(t, ignored, IgnoredMovement{reason: OccursInPast, movement: tooEarly})
				})

				it("contains movements that were scheduled after the halt", func() {
					assert.Contains(t, ignored, IgnoredMovement{reason: OccursAfterHalt, movement: tooLate})
				})

				it("doesn't contain any events that were scheduled", func() {
					assert.NotContains(t, ignored, IgnoredMovement{reason: OccursInPast, movement: goldilocks})
					assert.NotContains(t, ignored, IgnoredMovement{reason: OccursAfterHalt, movement: goldilocks})
				})
			})
		})

		describe.Pend("AddScheduleListener()", func() {
			it("adds a registered listener", func() {

			})
		})

		describe("helper funcs", func() {
			describe("occursAtToKey()", func() {
				it.Before(func() {
					movement = NewMovement(time.Unix(0, 111000111), fromStock, toStock)
				})

				it("returns the OccursAt() as a string", func() {
					key, err := occursAtToKey(movement)
					assert.NoError(t, err)
					assert.Equal(t, "111000111", key)
				})
			})

			describe("leftMovementIsEarlier()", func() {
				var earlier, later Movement

				it.Before(func() {
					earlier = NewMovement(time.Unix(111, 0), fromStock, toStock)
					later = NewMovement(time.Unix(999, 0), fromStock, toStock)
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
	})
}
