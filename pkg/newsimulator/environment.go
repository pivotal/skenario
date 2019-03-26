package newsimulator

import (
	"fmt"
	"strconv"
	"time"

	"k8s.io/client-go/tools/cache"
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

	runningScenario ThroughStock
	haltedScenario  ThroughStock

	futureMovements *cache.Heap
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
		err := env.futureMovements.Add(movement)
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

func (env *environment) Run() (completed []CompletedMovement, ignored []IgnoredMovement, err error) {
	// TODO: totally fake while tests get filled out
	// TODO: as this won't preserve order
	list := env.futureMovements.List()
	for _, l := range list {
		mv := l.(Movement)

		// TODO: handle nils and errors
		mv.To().Add(mv.From().Remove())

		env.completed = append(env.completed, CompletedMovement{movement: mv})
	}

	return env.completed, env.ignored, nil
}

func NewEnvironment(startAt time.Time, runFor time.Duration) Environment {
	heap := cache.NewHeap(occursAtToKey, leftMovementIsEarlier)

	env := &environment{
		current: startAt.Add(-1 * time.Nanosecond), // make temporary space for the Start Scenario movement
		startAt: startAt,
		haltAt:  startAt.Add(runFor),

		runningScenario: NewThroughStock("RunningScenario", "Scenario"),
		haltedScenario:  NewThroughStock("HaltedScenario", "Scenario"),
		futureMovements: heap,
		completed:       make([]CompletedMovement, 0),
		ignored:         make([]IgnoredMovement, 0),
	}

	env = setupScenarioMovements(env, startAt, env.haltAt, env.runningScenario, env.haltedScenario)
	env.current = startAt // restore proper starting time

	return env
}

func occursAtToKey(movement interface{}) (key string, err error) {
	mv := movement.(Movement)
	return strconv.FormatInt(mv.OccursAt().UnixNano(), 10), nil
}

func leftMovementIsEarlier(left interface{}, right interface{}) bool {
	l := left.(Movement)
	r := right.(Movement)

	return l.OccursAt().Before(r.OccursAt())
}

func setupScenarioMovements(env *environment, startAt time.Time, haltAt time.Time, runningScenario, haltedScenario ThroughStock) *environment {
	scenarioEntity := NewEntity("Scenario", "Scenario")
	err := haltedScenario.Add(scenarioEntity)
	if err != nil {
		panic(fmt.Errorf("could not add Scenario entity to haltedScenario: %s", err.Error()))
	}

	startMovement := NewMovement(startAt, haltedScenario, runningScenario, "Start scenario")
	haltMovement := NewMovement(haltAt, runningScenario, haltedScenario, "Halt scenario")

	env.AddToSchedule(startMovement)
	env.AddToSchedule(haltMovement)

	return env
}
