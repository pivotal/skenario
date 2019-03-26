package newsimulator

import (
	"fmt"
	"time"
)

const (
	OccursInPast    = "ScheduledToOccurInPast"
	OccursAfterHalt = "ScheduledToOccurAfterHalt"
)

type Environment interface {
	AddToSchedule(movement Movement) (added bool)
	Run() (completed []CompletedMovement, ignored []IgnoredMovement, err error)
}

type CompletedMovement struct {
	movement Movement
}

type IgnoredMovement struct {
	reason   string
	movement Movement
}

type environment struct {
	current time.Time
	startAt time.Time
	haltAt  time.Time

	beforeScenario  ThroughStock
	runningScenario ThroughStock
	haltedScenario  ThroughStock

	futureMovements MovementPriorityQueue
	completed       []CompletedMovement
	ignored         []IgnoredMovement
}

func (env *environment) AddToSchedule(movement Movement) (added bool) {
	occursAfterCurrent := movement.OccursAt().After(env.current)
	occursBeforeStart := movement.OccursAt().Before(env.startAt)
	occursBeforeHalt := movement.OccursAt().Before(env.haltAt)
	occursAtHalt := movement.OccursAt().Equal(env.haltAt)

	schedulable := occursAfterCurrent && (occursBeforeHalt || occursAtHalt)
	if schedulable {
		err := env.futureMovements.EnqueueMovement(movement)
		if err != nil {
			panic(fmt.Errorf("could not add '%#v' to future movements: %s", movement, err.Error()))
		}
	} else if !occursAfterCurrent || occursBeforeStart {
		env.ignored = append(env.ignored, IgnoredMovement{
			reason:   OccursInPast,
			movement: movement,
		})
	} else if !occursBeforeHalt {
		env.ignored = append(env.ignored, IgnoredMovement{
			reason:   OccursAfterHalt,
			movement: movement,
		})
	}

	return schedulable
}

func (env *environment) Run() ([]CompletedMovement, []IgnoredMovement, error) {
	for {
		movement, err, closed := env.futureMovements.DequeueMovement()
		if err != nil {
			return nil, nil, err
		}

		if closed {
			break
		}

		// TODO: handle nils and errors
		movement.To().Add(movement.From().Remove())

		env.completed = append(env.completed, CompletedMovement{movement: movement})
	}

	return env.completed, env.ignored, nil
}

func NewEnvironment(startAt time.Time, runFor time.Duration) Environment {
	pqueue := NewMovementPriorityQueue()
	return newEnvironment(startAt, runFor, pqueue)
}

func newEnvironment(startAt time.Time, runFor time.Duration, pqueue MovementPriorityQueue) *environment {
	beforeStock := NewThroughStock("BeforeScenario", "Scenario")
	runningStock := NewThroughStock("RunningScenario", "Scenario")
	haltingStock := NewHaltingSink("HaltedScenario", "Scenario", pqueue)

	env := &environment{
		current: startAt.Add(-1 * time.Nanosecond), // make temporary space for the Start Scenario movement
		startAt: startAt,
		haltAt:  startAt.Add(runFor),

		beforeScenario:  beforeStock,
		runningScenario: runningStock,
		haltedScenario:  haltingStock,
		futureMovements: pqueue,
		completed:       make([]CompletedMovement, 0),
		ignored:         make([]IgnoredMovement, 0),
	}

	env = setupScenarioMovements(env, startAt, env.haltAt, env.beforeScenario, env.runningScenario, env.haltedScenario)
	env.current = startAt // restore proper starting time

	return env
}

func setupScenarioMovements(env *environment, startAt time.Time, haltAt time.Time, beforeScenario, runningScenario, haltedScenario ThroughStock) *environment {
	scenarioEntity := NewEntity("Scenario", "Scenario")
	err := beforeScenario.Add(scenarioEntity)
	if err != nil {
		panic(fmt.Errorf("could not add Scenario entity to haltedScenario: %s", err.Error()))
	}

	startMovement := NewMovement(startAt, beforeScenario, runningScenario, "Start scenario")
	haltMovement := NewMovement(haltAt, runningScenario, haltedScenario, "Halt scenario")

	env.AddToSchedule(startMovement)
	env.AddToSchedule(haltMovement)

	return env
}
