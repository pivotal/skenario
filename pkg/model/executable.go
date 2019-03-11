package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	StateCold                   = "DeadCold"
	StateDiskPulling            = "Pulling"
	StateDiskWarm               = "DiskWarm"
	StateLaunchingFromDisk      = "LaunchingFromDisk"
	StatePageCacheWarm          = "PageCacheWarm"
	StateLaunchingFromPageCache = "LaunchingFromPageCache"
	StateLiveProcess            = "LiveProcess"

	beginPulling        = "begin_pulling"
	finishPulling       = "finish_pulling"
	launchFromDisk      = "launch_from_disk"
	launchFromPageCache = "launch_from_page_cache"
	finishLaunching     = "finish_launching"
	killProcess         = "kill_process"
)

var (
	evtBeginPulling        = fsm.EventDesc{Name: beginPulling, Src: []string{StateCold}, Dst: StateDiskPulling}
	evtFinishPulling       = fsm.EventDesc{Name: finishPulling, Src: []string{StateDiskPulling}, Dst: StateDiskWarm}
	evtLaunchFromDisk      = fsm.EventDesc{Name: launchFromDisk, Src: []string{StateDiskWarm}, Dst: StateLaunchingFromDisk}
	evtLaunchFromPageCache = fsm.EventDesc{Name: launchFromPageCache, Src: []string{StatePageCacheWarm}, Dst: StateLaunchingFromPageCache}
	evtFinishLaunching     = fsm.EventDesc{Name: finishLaunching, Src: []string{StateLaunchingFromDisk, StateLaunchingFromPageCache}, Dst: StateLiveProcess}
	evtKillProcess         = fsm.EventDesc{Name: killProcess, Src: []string{StateLiveProcess}, Dst: StatePageCacheWarm}
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
		launchFromPageCache: {
			EventName:   finishLaunching,
			Time:        t.Add(100 * time.Millisecond),
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

	var kickoffEventName string
	switch e.fsm.Current() {
	case StateCold:
		kickoffEventName = beginPulling
	case StateDiskWarm:
		kickoffEventName = launchFromDisk
	case StatePageCacheWarm:
		kickoffEventName = launchFromPageCache
	default:
		fmt.Println("Info: ignored state", e.fsm.Current(), "and set to", StateCold)
		e.fsm.SetState(StateCold)
		kickoffEventName = beginPulling
	}

	env.Schedule(&simulator.Event{
		Time:        env.Time().Add(time.Duration(r) * time.Millisecond),
		EventName:   kickoffEventName,
		AdvanceFunc: e.Advance,
	})
}

func NewExecutable(name, initialState string) *Executable {
	return &Executable{
		name: name,
		fsm: fsm.NewFSM(
			initialState,
			fsm.Events{
				evtBeginPulling,
				evtFinishPulling,
				evtLaunchFromDisk,
				evtFinishLaunching,
				evtLaunchFromPageCache,
				evtKillProcess,
			},
			fsm.Callbacks{},
		),
	}
}
