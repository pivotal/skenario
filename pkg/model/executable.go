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
	finishLaunching     = "finish_launching_process"
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
	name     simulator.ProcessIdentity
	fsm      *fsm.FSM
	env      *simulator.Environment
	replicas []*RevisionReplica
}

func (e *Executable) Identity() simulator.ProcessIdentity {
	return e.name
}

func (e *Executable) OnOccurrence(event *simulator.Event) (result simulator.TransitionResult) {
	var nextExecEvtName string
	var nextExecEvtTime time.Time

	switch event.EventName {
	case beginPulling:
		nextExecEvtName = finishPulling
		nextExecEvtTime = event.OccursAt.Add(90 * time.Second)
	case finishPulling:
		nextExecEvtName = launchFromDisk
		nextExecEvtTime = event.OccursAt.Add(1 * time.Second)
	case launchFromDisk:
		nextExecEvtName = finishLaunching
		nextExecEvtTime = event.OccursAt.Add(10 * time.Second)
	case launchFromPageCache:
		nextExecEvtName = finishLaunching
		nextExecEvtTime = event.OccursAt.Add(100 * time.Millisecond)
	case killProcess:
		if len(e.replicas) > 1 { // if there will still be running replicas after termination
			return simulator.TransitionResult{FromState: e.fsm.Current(), ToState: e.fsm.Current()} // do nothing
		}
	}

	if event.EventName != killProcess && event.EventName != finishLaunching {
		execEvt := &simulator.Event{
			EventName: nextExecEvtName,
			OccursAt:  nextExecEvtTime,
			Subject:   e,
		}

		e.env.Schedule(execEvt)
	}

	current := e.fsm.Current()
	err := e.fsm.Event(event.EventName)
	if err != nil {
		panic(err.Error())
	}

	return simulator.TransitionResult{FromState: current, ToState: e.fsm.Current()}
}

func (e *Executable) OnSchedule(event *simulator.Event) {
	if event.EventName == finishTerminatingReplica {
		e.env.Schedule(&simulator.Event{
			EventName: killProcess,
			OccursAt:  event.OccursAt.Add(-100 * time.Millisecond),
			Subject:   e,
		})
	}
}

func (e *Executable) Run(startingAt time.Time) {
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

	e.env.Schedule(&simulator.Event{
		OccursAt:  startingAt.Add(time.Duration(r) * time.Millisecond),
		EventName: kickoffEventName,
		Subject:   e,
	})
}

func (e *Executable) AddRevisionReplica(replica *RevisionReplica) {
	e.replicas = append(e.replicas, replica)

	e.env.ListenForScheduling(replica.Identity(), finishTerminatingReplica, e)
}

func NewExecutable(name simulator.ProcessIdentity, initialState string, env *simulator.Environment) *Executable {
	return &Executable{
		name: name,
		env: env,
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
