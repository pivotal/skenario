package newmodel

import (
	"time"

	"knative-simulator/pkg/newsimulator"

	"github.com/knative/serving/pkg/autoscaler"
)

type KnativeAutoscaler interface {
}

type knativeAutoscaler struct {
	env        newsimulator.Environment
	tickTock   *tickTock
	autoscaler *autoscaler.Autoscaler
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
