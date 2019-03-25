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
	Run() (results []CompletedMovement, movements []IgnoredMovement, err error)
}

type CompletedMovement struct {
	movement Movement
}

type IgnoredMovement struct {
	reason   string
	movement Movement
}

type environment struct {
	simulationTime time.Time
	haltTime       time.Time

	futureMovements *cache.Heap
	completed       []CompletedMovement
	ignored         []IgnoredMovement
}

func (env *environment) AddToSchedule(movement Movement) (added bool) {
	occursAfterCurrent := movement.OccursAt().After(env.simulationTime)
	occursBeforeHalt := movement.OccursAt().Before(env.haltTime)
	schedulable := occursAfterCurrent && occursBeforeHalt
	if schedulable {
		err := env.futureMovements.Add(movement)
		if err != nil {
			panic(fmt.Errorf("could not add '%#v' to future movements: %s", movement, err.Error()))
		}
	} else if !occursAfterCurrent {
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

func (env *environment) Run() (results []CompletedMovement, movements []IgnoredMovement, err error) {
	// TODO: totally fake while tests get filled out
	// TODO: as this won't preserve order
	list := env.futureMovements.List()
	for _, l := range list {
		mv := l.(Movement)

		mv.To().Add(mv.From().Remove())

		env.completed = append(env.completed, CompletedMovement{movement: mv})
	}

	return env.completed, env.ignored, nil
}

func NewEnvironment(startAt time.Time, runFor time.Duration) Environment {
	heap := cache.NewHeap(occursAtToKey, leftMovementIsEarlier)

	return &environment{
		simulationTime: startAt,
		haltTime:       startAt.Add(runFor),

		futureMovements: heap,
		completed:       make([]CompletedMovement, 0),
		ignored:         make([]IgnoredMovement, 0),
	}
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
