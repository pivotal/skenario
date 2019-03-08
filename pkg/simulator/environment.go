package simulator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/tools/cache"
)

type Environment struct {
	futureEvents *cache.Heap
	simTime      time.Time
	endTime      time.Time
}

func NewEnvironment(begin time.Time, runFor time.Duration) *Environment {
	heap := cache.NewHeap(func(event interface{}) (key string, err error) {
		evt := event.(*Event)
		return strconv.FormatInt(evt.Time.UnixNano(), 10), nil
	}, func(leftEvent interface{}, rightEvent interface{}) bool {
		l := leftEvent.(*Event)
		r := rightEvent.(*Event)

		return l.Time.Before(r.Time)
	})

	env := &Environment{
		futureEvents: heap,
		simTime:      begin,
		endTime:      begin.Add(runFor),
	}

	termEvent := &Event{
		Time:        env.endTime,
		Description: "Termination event",
		AdvanceFunc: env.Terminate,
	}

	env.Schedule(termEvent)

	return env
}

func (env *Environment) Run() {
	fmt.Printf("[%d] Simulation begins\n", env.simTime.UnixNano())

	for {
		nextIface, err := env.futureEvents.Pop() // blocks until there is stuff to pop
		if err != nil && strings.Contains(err.Error(), "heap is closed") {
			return
		} else if err != nil {
			panic(err)
		}

		next := nextIface.(*Event)
		env.simTime = next.Time
		next.AdvanceFunc(next.Time, next.Description)
	}
}

func (env *Environment) Schedule(event *Event) {
	if event.Time.After(env.endTime) {
		fmt.Printf("Ignoring event scheduled after termination: [%d] %s\n", event.Time.UnixNano(), event.Description)
		return
	}

	err := env.futureEvents.Add(event)
	if err != nil {
		panic(err)
	}
}

func (env *Environment) Terminate(time time.Time, description string) {
	fmt.Printf("[%d] Reached termination event", time.UnixNano())
	env.futureEvents.Close()
}
