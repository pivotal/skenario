package simulator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/looplab/fsm"
	"k8s.io/client-go/tools/cache"
)

type Environment struct {
	futureEvents *cache.Heap
	simTime      time.Time
	startTime    time.Time
	endTime      time.Time

	fsm *fsm.FSM
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
		startTime:    begin,
		endTime:      begin.Add(runFor),
	}

	env.fsm = fsm.NewFSM(
		"STARTING",
		fsm.Events{
			{Name: "start", Src: []string{"STARTING"}, Dst: "RUNNING"},
			{Name: "terminate", Src: []string{"RUNNING"}, Dst: "TERMINATED"},
		},
		fsm.Callbacks{},
	)

	startEvent := &Event{
		Time:        env.startTime,
		EventName:   "start",
		AdvanceFunc: env.Start,
	}

	termEvent := &Event{
		Time:        env.endTime,
		EventName:   "terminate",
		AdvanceFunc: env.Terminate,
	}

	env.Schedule(startEvent)
	env.Schedule(termEvent)

	return env
}

func (env *Environment) Run() {
	//fmt.Printf("[%d] Simulation begins\n", env.simTime.UnixNano())

	for {
		nextIface, err := env.futureEvents.Pop() // blocks until there is stuff to pop
		if err != nil && strings.Contains(err.Error(), "heap is closed") {
			return
		} else if err != nil {
			panic(err)
		}

		next := nextIface.(*Event)
		env.simTime = next.Time
		procName, outcome := next.AdvanceFunc(next.Time, next.EventName)
		fmt.Printf("[%d] [%s] %s: %s\n", next.Time.UnixNano(), procName, next.EventName, outcome)
	}
}

func (env *Environment) Schedule(event *Event) {
	if event.Time.After(env.endTime) {
		fmt.Printf("Ignoring event scheduled after termination: [%d] %s\n", event.Time.UnixNano(), event.EventName)
		return
	}

	err := env.futureEvents.Add(event)
	if err != nil {
		panic(err)
	}
}

func (env *Environment) Start(time time.Time, description string) (identifier, outcome string) {
	return "Environment", "Started simulation"
}

func (env *Environment) Terminate(time time.Time, description string) (identifier, outcome string) {
	env.futureEvents.Close()
	return "Environment", "Reached termination event"
}
