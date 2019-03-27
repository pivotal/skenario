package newmodel

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "KnativeAutoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

type fakeEnvironment struct {
	movements []newsimulator.Movement
	listeners []newsimulator.MovementListener
}

func (fe *fakeEnvironment) AddToSchedule(movement newsimulator.Movement) (added bool) {
	fe.movements = append(fe.movements, movement)
	return true
}

func (fe *fakeEnvironment) AddMovementListener(listener newsimulator.MovementListener) error {
	fe.listeners = append(fe.listeners, listener)
	return nil
}

func (fe *fakeEnvironment) Run() (completed []newsimulator.CompletedMovement, ignored []newsimulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func testAutoscaler(t *testing.T, describe spec.G, it spec.S) {
	var envFake *fakeEnvironment
	startAt := time.Unix(0, 0)
	// runFor := 1 * time.Minute

	it.Before(func() {
		envFake = &fakeEnvironment{
			movements: make([]newsimulator.Movement, 0),
			listeners: make([]newsimulator.MovementListener, 0),
		}
	})

	describe("NewKnativeAutoscaler()", func() {
		it.Before(func() {
			_ = NewKnativeAutoscaler(envFake, startAt)
		})

		it("schedules a first calculation", func() {
			firstCalc := envFake.movements[0]
			assert.Equal(t, newsimulator.MovementKind("waiting_to_calculating"), firstCalc.Kind())
		})

		it.Pend("registers itself as a MovementListener", func() {

		})
	})

	describe("OnMovement()", func() {
		var subject KnativeAutoscaler
		var waitToCalcMovement newsimulator.Movement
		var ttStock *tickTock

		it.Before(func() {
			subject = NewKnativeAutoscaler(envFake, startAt)

			ttStock = &tickTock{}
			waitToCalcMovement = newsimulator.NewMovement("waiting_to_calculating", time.Now(), ttStock, ttStock, "test movement note")

			err := subject.OnMovement(waitToCalcMovement)
			assert.NoError(t, err)
		})

		it("schedules movements for the next wait/calculate cycle", func() {
			calcInit := envFake.movements[0]
			assert.Equal(t, newsimulator.MovementKind("waiting_to_calculating"), calcInit.Kind())

			wait := envFake.movements[1]
			assert.Equal(t, newsimulator.MovementKind("calculating_to_waiting"), wait.Kind())

			calc := envFake.movements[2]
			assert.Equal(t, newsimulator.MovementKind("waiting_to_calculating"), calc.Kind())
		})

		it.Pend("triggers the autoscaler calculation", func() {

		})
	})

	describe("tickTock stock", func() {
		ttStock := &tickTock{}

		it.Before(func() {
			_ = NewKnativeAutoscaler(envFake, startAt)
		})

		describe("Name()", func() {
			it("is called 'KnativeAutoscaler Stock'", func() {
				assert.Equal(t, ttStock.Name(), newsimulator.StockName("KnativeAutoscaler Stock"))
			})
		})

		describe("KindStocked()", func() {
			it("accepts Knative Autoscalers", func() {
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
		})
	})
}
