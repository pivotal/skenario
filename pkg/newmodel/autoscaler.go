package newmodel

import (
	"time"

	"knative-simulator/pkg/newsimulator"

	"github.com/knative/serving/pkg/autoscaler"
)

type KnativeAutoscaler interface {
	newsimulator.MovementListener
}

type knativeAutoscaler struct {
	env        newsimulator.Environment
	tickTock   *tickTock
	autoscaler *autoscaler.Autoscaler
}

func (kas *knativeAutoscaler) OnMovement(movement newsimulator.Movement) error {
	if movement.Kind() == "waiting_to_calculating" {
		waitingMovement := newsimulator.NewMovement(
			"calculating_to_waiting",
			movement.OccursAt().Add(1*time.Nanosecond),
			kas.tickTock,
			kas.tickTock,
			"Autoscaler calculating",
			)

		calculatingMovement := newsimulator.NewMovement(
			"waiting_to_calculating",
			movement.OccursAt().Add(2*time.Second),
			kas.tickTock,
			kas.tickTock,
			"Autoscaler waiting",
			)

		kas.env.AddToSchedule(waitingMovement)
		kas.env.AddToSchedule(calculatingMovement)
	}

	return nil
}

func NewKnativeAutoscaler(env newsimulator.Environment, startAt time.Time) KnativeAutoscaler {
	kas := &knativeAutoscaler{
		env:        env,
		tickTock:   &tickTock{},
		autoscaler: nil,
	}

	firstCalculation := newsimulator.NewMovement(
		"waiting_to_calculating",
		startAt.Add(2*time.Second),
		kas.tickTock,
		kas.tickTock,
		"Autoscaler calculating",
	)

	env.AddToSchedule(firstCalculation)
	err := env.AddMovementListener(kas)
	if err != nil {
		panic(err.Error())
	}

	return kas
}

type tickTock struct {
	asEntity newsimulator.Entity
}

func (tt *tickTock) Name() newsimulator.StockName {
	return "KnativeAutoscaler Stock"
}

func (tt *tickTock) KindStocked() newsimulator.EntityKind {
	return newsimulator.EntityKind("KnativeAutoscaler")
}

func (tt *tickTock) Count() uint64 {
	return 1
}

func (tt *tickTock) Remove() newsimulator.Entity {
	return tt.asEntity
}

func (tt *tickTock) Add(entity newsimulator.Entity) error {
	tt.asEntity = entity

	return nil
}
