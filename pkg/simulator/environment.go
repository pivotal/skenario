package simulator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/looplab/fsm"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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
		"SimulationStarting",
		fsm.Events{
			{Name: "start_simulation", Src: []string{"SimulationStarting"}, Dst: "SimulationRunning"},
			{Name: "terminate_simulation", Src: []string{"SimulationRunning"}, Dst: "SimulationTerminated"},
		},
		fsm.Callbacks{},
	)

	startEvent := &Event{
		Time:        env.startTime,
		EventName:   "start_simulation",
		AdvanceFunc: env.Start,
	}

	termEvent := &Event{
		Time:        env.endTime,
		EventName:   "terminate_simulation",
		AdvanceFunc: env.Terminate,
	}

	env.Schedule(startEvent)
	env.Schedule(termEvent)

	return env
}

func (env *Environment) Run() {
	printer := message.NewPrinter(language.AmericanEnglish)
	printer.Printf("%-21s   %-12s  %-26s   %-22s -->  %-25s %s\n", "TIME", "IDENTIFIER", "EVENT", "FROM STATE", "TO STATE", "NOTE")
	for {
		nextIface, err := env.futureEvents.Pop() // blocks until there is stuff to pop
		if err != nil && strings.Contains(err.Error(), "heap is closed") {
			return
		} else if err != nil {
			panic(err)
		}

		next := nextIface.(*Event)
		env.simTime = next.Time
		procName, fromState, toState, note := next.AdvanceFunc(next.Time, next.EventName)
		printer.Printf("[%21d] %-12s  %-26s   %-22s -->  %-25s %s\n", next.Time.UnixNano(), procName, next.EventName, fromState, toState, note)
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

func (env *Environment) Start(time time.Time, description string) (identifier, fromState, toState, note string) {
	return "Environment", "SimulationStarting", "SimulationRunning", "Started simulation"
}

func (env *Environment) Terminate(time time.Time, description string) (identifier, fromState, toState, note string) {
	env.futureEvents.Close()
	return "Environment", "SimulationRunning", "SimulationTerminated", "Reached termination event"
}

func (env *Environment) Time() time.Time {
	return env.simTime
}