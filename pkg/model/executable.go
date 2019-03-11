package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	cold                   = "DeadCold"
	diskPulling            = "Pulling"
	diskWarm               = "DiskWarm"
	launchingFromDisk      = "LaunchingFromDisk"
	pageCacheWarm          = "PageCacheWarm"
	launchingFromPageCache = "LaunchingFromPageCache"
	liveProcess            = "LiveProcess"

	beginPulling    = "begin_pulling"
	finishPulling   = "finish_pulling"
	launchFromDisk  = "launch_from_disk"
	finishLaunching = "finish_launching"
	killProcess     = "kill_process"
)

var (
	EvtBeginPulling            = fsm.EventDesc{Name: beginPulling, Src: []string{cold}, Dst: diskPulling}
	EvtFinishPulling           = fsm.EventDesc{Name: finishPulling, Src: []string{diskPulling}, Dst: diskWarm}
	EvtLaunchFromDisk          = fsm.EventDesc{Name: launchFromDisk, Src: []string{diskWarm}, Dst: launchingFromDisk}
	EvtFinishLaunchingFromDisk = fsm.EventDesc{Name: finishLaunching, Src: []string{launchingFromDisk, launchingFromPageCache}, Dst: liveProcess}
	EvtKillProcess             = fsm.EventDesc{Name: killProcess, Src: []string{liveProcess}, Dst: pageCacheWarm}
)

type Executable struct {
	name string
	fsm  *fsm.FSM
	env  *simulator.Environment
}

func (e *Executable) Advance(t time.Time, eventName string) (identifier, outcome string) {
	nextEventMap := map[string]simulator.Event{
		beginPulling: {
			EventName:   finishPulling,
			Time:        t.Add(90 * time.Second),
			AdvanceFunc: e.Advance,
		},
		finishPulling: {
			EventName:   launchFromDisk,
			Time:        t.Add(1 * time.Second),
			AdvanceFunc: e.Advance,
		},
		launchFromDisk: {
			EventName:   finishLaunching,
			Time:        t.Add(10 * time.Second),
			AdvanceFunc: e.Advance,
		},
		finishLaunching: {
			EventName:   killProcess,
			Time:        t.Add(30 * time.Second),
			AdvanceFunc: e.Advance,
		},
	}

	if eventName != killProcess {
		evt := nextEventMap[eventName]
		e.env.Schedule(&evt)
	}

	current := e.fsm.Current()
	err := e.fsm.Event(eventName)
	if err != nil {
		panic(err.Error())
	}

	return e.name, fmt.Sprintf("%s --> %s", current, e.fsm.Current())
}

func (e *Executable) Run(env *simulator.Environment) {
	e.env = env

	r := rand.Intn(100)

	env.Schedule(&simulator.Event{
		Time:        env.Time().Add(time.Duration(r) * time.Millisecond),
		EventName:   EvtBeginPulling.Name,
		AdvanceFunc: e.Advance,
	})
}

func NewExecutable(name string) *Executable {
	return &Executable{
		name: name,
		fsm: fsm.NewFSM(
			cold,
			fsm.Events{
				EvtBeginPulling,
				EvtFinishPulling,
				EvtLaunchFromDisk,
				EvtFinishLaunchingFromDisk,
				EvtKillProcess,
			},
			fsm.Callbacks{},
		),
	}
}
