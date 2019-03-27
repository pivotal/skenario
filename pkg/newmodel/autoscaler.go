package newmodel

import (
	"time"

	"knative-simulator/pkg/newsimulator"

	"github.com/knative/serving/pkg/autoscaler"
)

const (
	MvWaitingToCalculating newsimulator.MovementKind = "autoscaler_wait"
	MvCalculatingToWaiting newsimulator.MovementKind = "autoscaler_calc"
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
	if movement.Kind() == MvWaitingToCalculating {
		waitingMovement := newsimulator.NewMovement(
			MvCalculatingToWaiting,
			movement.OccursAt().Add(1*time.Nanosecond),
			kas.tickTock,
			kas.tickTock,
			"",
			)

		calculatingMovement := newsimulator.NewMovement(
			MvWaitingToCalculating,
			movement.OccursAt().Add(2*time.Second),
			kas.tickTock,
			kas.tickTock,
			"",
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
		MvWaitingToCalculating,
		startAt.Add(2001*time.Millisecond),
		kas.tickTock,
		kas.tickTock,
		"First calculation",
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
	return "Autoscaler ticktock"
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
