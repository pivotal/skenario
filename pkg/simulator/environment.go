package simulator

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"k8s.io/client-go/tools/cache"
)

type Environment struct {
	futureEvents        *cache.Heap
	ignoredEvents       []Event
	registeredListeners map[ProcessIdentity]map[EventName]SchedulingListener

	simTime   time.Time
	startTime time.Time
	endTime   time.Time
}

func NewEnvironment(begin time.Time, runFor time.Duration) *Environment {
	heap := cache.NewHeap(func(event interface{}) (key string, err error) {
		evt := event.(Event)
		return strconv.FormatInt(evt.OccursAt().UnixNano(), 10), nil
	}, func(leftEvent interface{}, rightEvent interface{}) bool {
		l := leftEvent.(Event)
		r := rightEvent.(Event)

		return l.OccursAt().Before(r.OccursAt())
	})

	env := &Environment{
		futureEvents:        heap,
		ignoredEvents:       make([]Event, 0),
		registeredListeners: make(map[ProcessIdentity]map[EventName]SchedulingListener),
		simTime:             begin,
		startTime:           begin,
		endTime:             begin.Add(runFor),
	}

	startEvent := NewGeneralEvent(
		"start_simulation",
		env.startTime,
		env,
	)

	termEvent := NewGeneralEvent(
		"terminate_simulation",
		env.endTime,
		env,
	)

	env.Schedule(startEvent)
	env.Schedule(termEvent)

	return env
}

func (env *Environment) Run() {
	printer := message.NewPrinter(language.AmericanEnglish)
	printer.Printf("%20s    %-18s  %-26s    %-25s -->  %-25s    %s\n", "TIME (ns)", "IDENTIFIER", "EVENT", "FROM STATE", "TO STATE", "NOTE")
	printer.Println("---------------------------------------------------------------------------------------------------------------------------------------------------------------")
	for {
		nextIface, err := env.futureEvents.Pop() // blocks until there is stuff to pop
		if err != nil && strings.Contains(err.Error(), "heap is closed") {
			for _, e := range env.ignoredEvents {
				printer.Println("---------------------------------------------------------------------------------------------------------------------------------------------------------------")
				printer.Println("Ignored events were ignored as they were scheduled after termination:")
				printer.Printf("%20d    %-18s  %-26s\n", e.OccursAt().UnixNano(), "", e.Name())
			}
			return
		} else if err != nil {
			panic(err)
		}

		evt := nextIface.(Event)
		switch evt.Kind() {
		case EventGeneral:
			next := evt.(GeneralEvent)
			env.simTime = next.OccursAt()
			subject := next.Subject().(Process)
			result := subject.OnOccurrence(next)
			printer.Printf("G %20d    %-18s  %-26s    %-25s -->  %-25s    %s\n", next.OccursAt().UnixNano(), next.SubjectIdentity(), next.Name(), result.FromState, result.ToState, result.Note)

		case EventMovement:
			next := evt.(StockMovementEvent)

			env.simTime = next.OccursAt()
			subject := next.Subject().(Stockable)
			next.From().UpdateStock(next)
			next.To().UpdateStock(next)
			result := subject.OnMovement(next)
			printer.Printf("M %20d    %-18s  %-26s    %-25s -->  %-25s    %s\n", next.OccursAt().UnixNano(), next.SubjectIdentity(), next.Name(), result.FromStock.Identity(), result.ToStock.Identity(), result.Note)
		}
	}
}

func (env *Environment) Schedule(event Event) {
	if evtNameMap, ok := env.registeredListeners[event.SubjectIdentity()]; ok {
		if listener, ok := evtNameMap[event.Name()]; ok {
			listener.OnSchedule(event)
		}
	}

	if event.OccursAt().After(env.endTime) {
		env.ignoredEvents = append(env.ignoredEvents, event)
		return
	}

	err := env.futureEvents.Add(event)
	if err != nil {
		panic(err)
	}
}

func (env *Environment) Identity() ProcessIdentity {
	return "Environment"
}

func (env *Environment) OnOccurrence(event Event) (result StateTransitionResult) {
	switch event.Name() {
	case "start_simulation":
		return StateTransitionResult{FromState: "SimulationStarting", ToState: "SimulationRunning", Note: "Started simulation"}
	case "terminate_simulation":
		env.futureEvents.Close()
		return StateTransitionResult{FromState: "SimulationRunning", ToState: "SimulationTerminated", Note: "Reached termination event"}
	default:
		panic("Unknown event for Environment")
	}
}

func (env *Environment) ListenForScheduling(subjectIdentity ProcessIdentity, eventName EventName, listener SchedulingListener) {
	if _, ok := env.registeredListeners[subjectIdentity]; ok {
		env.registeredListeners[subjectIdentity][eventName] = listener
	} else {
		env.registeredListeners[subjectIdentity] = make(map[EventName]SchedulingListener)
		env.registeredListeners[subjectIdentity][eventName] = listener
	}
}

func (env *Environment) Time() time.Time {
	return env.simTime
}
