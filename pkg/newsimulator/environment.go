package newsimulator

import (
	"fmt"
	"strings"
	"time"
)

const (
	OccursInPast                            = "ScheduledToOccurInPast"
	OccursAfterHalt                         = "ScheduledToOccurAfterHalt"
	OccursSimultaneouslyWithAnotherMovement = "ScheduleCollidesWithAnotherMovement"
)

type Environment interface {
	AddToSchedule(movement Movement) (added bool)
	AddMovementListener(listener MovementListener) error
	Run() (completed []CompletedMovement, ignored []IgnoredMovement, err error)
}

type CompletedMovement struct {
	Movement Movement
}

type IgnoredMovement struct {
	Reason   string
	Movement Movement
}

type environment struct {
	current time.Time
	startAt time.Time
	haltAt  time.Time

	beforeScenario  ThroughStock
	runningScenario ThroughStock
	haltedScenario  ThroughStock

	futureMovements   MovementPriorityQueue
	completed         []CompletedMovement
	ignored           []IgnoredMovement
	movementListeners []MovementListener
}

func (env *environment) AddToSchedule(movement Movement) (added bool) {
	occursAfterCurrent := movement.OccursAt().After(env.current)
	occursAfterHalt := movement.OccursAt().After(env.haltAt)

	schedulable := occursAfterCurrent && !occursAfterHalt
	if schedulable {
		err := env.futureMovements.EnqueueMovement(movement)
		if err != nil {
			if strings.Contains(err.Error(), "there is already another movement scheduled at that time") {
				env.ignored = append(env.ignored, IgnoredMovement{
					Reason:   OccursSimultaneouslyWithAnotherMovement,
					Movement: movement,
				})

				return false
			} else {
				panic(fmt.Errorf("unknown error meant '%#v' was not added future movements: %s", movement, err.Error()))
			}
		}
	} else if !occursAfterCurrent {
		env.ignored = append(env.ignored, IgnoredMovement{
			Reason:   OccursInPast,
			Movement: movement,
		})
	} else if occursAfterHalt {
		env.ignored = append(env.ignored, IgnoredMovement{
			Reason:   OccursAfterHalt,
			Movement: movement,
		})
	}

	return schedulable
}

func (env *environment) AddMovementListener(listener MovementListener) error {
	env.movementListeners = append(env.movementListeners, listener)
	return nil
}

func (env *environment) Run() ([]CompletedMovement, []IgnoredMovement, error) {
	for {
		var err error

		movement, err, closed := env.futureMovements.DequeueMovement()
		if err != nil {
			return nil, nil, err
		}

		if closed {
			break
		}

		for _, ml := range env.movementListeners {
			err = ml.OnMovement(movement)
			if err != nil {
				// TODO: panic might be overkill
				panic(err.Error())
			}
		}

		// TODO: handle nils and errors
		movement.To().Add(movement.From().Remove())

		env.completed = append(env.completed, CompletedMovement{Movement: movement})
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

		beforeScenario:    beforeStock,
		runningScenario:   runningStock,
		haltedScenario:    haltingStock,
		futureMovements:   pqueue,
		completed:         make([]CompletedMovement, 0),
		ignored:           make([]IgnoredMovement, 0),
		movementListeners: make([]MovementListener, 0),
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

	startMovement := NewMovement("start_to_running", startAt, beforeScenario, runningScenario)
	startMovement.AddNote("Start scenario")
	haltMovement := NewMovement("running_to_halted", haltAt, runningScenario, haltedScenario)
	haltMovement.AddNote("Halt scenario")

	env.AddToSchedule(startMovement)
	env.AddToSchedule(haltMovement)

	return env
}
