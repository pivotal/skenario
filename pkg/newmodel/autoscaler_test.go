package newmodel

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"knative-simulator/pkg/newsimulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "KnativeAutoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

type mockEnvironment struct {
	mock.Mock
}

func (me *mockEnvironment) AddToSchedule(movement newsimulator.Movement) (added bool) {
	me.Called(movement)
	return true
}

func (me *mockEnvironment) Run() (completed []newsimulator.CompletedMovement, ignored []newsimulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func testAutoscaler(t *testing.T, describe spec.G, it spec.S) {
	// var subject KnativeAutoscaler
	var mockEnv *mockEnvironment
	startAt := time.Unix(0, 0)
	// runFor := 1 * time.Minute

	it.Before(func() {
		mockEnv = new(mockEnvironment)
		mockEnv.On("AddToSchedule", mock.Anything).Return(true) // TODO: I'd rather not use mock.Anything
	})

	describe("NewKnativeAutoscaler()", func() {
		it.Before(func() {
			_ = NewKnativeAutoscaler(mockEnv, startAt)
		})

		it("schedules a first calculation", func() {
			mockEnv.AssertExpectations(t)
		})
	})

	describe("tickTock stock", func() {
		it.Before(func() {
			_ = NewKnativeAutoscaler(mockEnv, startAt)
		})
		ttStock := &tickTock{}

		describe("Name()", func() {
			it("is called 'KnativeAutoscaler Stock'", func() {
				assert.Equal(t, ttStock.Name(), newsimulator.StockName("KnativeAutoscaler Stock"))
			})
		})

		describe("KindStocked()", func() {
			it("accepts Autoscalers", func() {
				assert.Equal(t, ttStock.KindStocked(), newsimulator.EntityKind("KnativeAutoscaler"))
			})
		})

		describe("Count()", func() {
			it("always has 1 entity stocked", func() {
				assert.Equal(t, ttStock.Count(), uint64(1))

				err := ttStock.Add(newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler")))
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Count(), uint64(1))
			})
		})

		describe("Remove()", func() {
			it("gives back the one KnativeAutoscaler", func() {
				entity := newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Remove(), entity)
			})
		})

		describe("Add()", func() {
			it("adds the entity if it's not already set", func() {
				assert.Nil(t, ttStock.asEntity)

				entity := newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.asEntity, entity)
			})

			it("schedules an event for the next calculation", func() {
				mockEnv.On("AddToSchedule", mock.Anything).Return(true) // TODO: I'd rather not use mock.Anything

				err := ttStock.Add(ttStock.asEntity)
				assert.NoError(t, err)

				mockEnv.AssertCalled(t, "AddToSchedule", newsimulator.NewMovement(
					"waiting_to_calculating",
					startAt.Add(2*time.Second),
					ttStock,
					ttStock,
					"Autoscaler calculating",
				))

				mockEnv.AssertNumberOfCalls(t, "AddToSchedule", 2)
			})

			it.Pend("triggers the autoscaler calculation", func() {

			})
		})
	})
}
