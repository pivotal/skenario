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

// We hand-roll the echo source stock, otherwise the compiler will use ThroughStock,
// leading to nil errors when we try to .Remove() a non-existent entry.
type echoSourceStockType struct {
	name   StockName
	kind   EntityKind
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

func TestEnvironment(t *testing.T) {
	spec.Run(t, "Environment spec", testEnvironment, spec.Report(report.Terminal{}))
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
	runFor := 555555 * time.Second

	it.Before(func() {
		subject = NewEnvironment(startTime, runFor)
		assert.NotNil(t, subject)

		fromStock = &echoSourceStockType{
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
			completed, ignored, err = subject.Run()
			assert.NoError(t, err)

			for _, c := range completed {
				completedNotes = append(completedNotes, c.movement.Note())
			}

			for _, i := range ignored {
				ignoredNotes = append(ignoredNotes, i.movement.Note())
				fmt.Println(i.reason)
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
		describe("the scheduled movement will occur during the simulation", func() {
			it("returns true", func() {
				movement = NewMovement(time.Unix(333333, 0), fromStock, toStock, "during sim test movement")
				assert.True(t, subject.AddToSchedule(movement))
			})
		})

		describe("the scheduled movement will occur at halt", func() {
			it("returns true", func() {
				movement = NewMovement(time.Unix(777777, 0), fromStock, toStock, "at halt test movement")
				assert.True(t, subject.AddToSchedule(movement))
			})
		})

		describe("the scheduled movement would occur after the simulation halts", func() {
			it("returns false", func() {
				movement = NewMovement(time.Unix(999999, 0), fromStock, toStock, "after halt test movement")
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the movement would occur before the current simulation time", func() {
			it("returns false", func() {
				movement = NewMovement(time.Unix(111111, 0), fromStock, toStock, "before simulation time test movement")
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe("the movement would occur at the current simulation time", func() {
			it("returns false", func() {
				movement = NewMovement(time.Unix(222222, 0), fromStock, toStock, "at simulation time test movement")
				assert.False(t, subject.AddToSchedule(movement))
			})
		})

		describe.Pend("Schedule listeners", func() {
			it("calls OnSchedule() on registered listeners", func() {

			})
		})
	}, spec.Nested())

	describe.Pend("AddScheduleListener()", func() {
		it("adds a registered listener", func() {

		})
	}, spec.Nested())

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

				movement = NewMovement(time.Unix(333333, 0), fromMock, toMock, "test movement")

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

		describe("results", func() {
			describe("completed movements", func() {
				var first, second Movement
				var completed []CompletedMovement

				it.Before(func() {
					var err error

					subject = NewEnvironment(startTime, runFor)

					first = NewMovement(time.Unix(333333, 0), fromStock, toStock, "first test movement")
					second = NewMovement(time.Unix(444444, 0), fromStock, toStock, "second test movement")

					subject.AddToSchedule(first)
					subject.AddToSchedule(second)
					completed, _, err = subject.Run()

					assert.NoError(t, err)
				})

				it("contains the correct number of completed movements", func() {
					assert.Len(t, completed, 4) // start scenario, halt scenario, first, second
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

					tooEarly = NewMovement(time.Unix(111111, 0), fromStock, toStock, "too early test movement")
					goldilocks = NewMovement(time.Unix(333333, 0), fromStock, toStock, "goldilocks test movement")
					tooLate = NewMovement(time.Unix(999999, 0), fromStock, toStock, "too late test movement")

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
	}, spec.Nested())

	describe("helper funcs", func() {
		describe("newEnvironment()", func() {
			var env *environment
			var mpq MovementPriorityQueue

			it.Before(func() {
				mpq = NewMovementPriorityQueue()
				env = newEnvironment(time.Unix(0, 0), time.Minute, mpq)
			})

			it("configures the halted scenario stock to use haltingStock", func() {
				assert.Equal(t, env.haltedScenario.Name(), StockName("HaltedScenario"))
				assert.IsType(t, &haltingSink{}, env.haltedScenario)
			})
		})
	}, spec.Nested())
}
