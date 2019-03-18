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
	name     string
	fsm      *fsm.FSM
	env      *simulator.Environment
	replicas []*RevisionReplica
}

func (e *Executable) Identity() string {
	return e.name
}

func (e *Executable) OnAdvance(event *simulator.Event) (result simulator.TransitionResult) {
	var nextExecEvtName, nextReplicaEvtName string
	var nextExecEvtTime, nextReplicaEvtTime time.Time

	switch event.EventName {
	case beginPulling:
		nextExecEvtName = finishPulling
		nextExecEvtTime = event.Time.Add(90 * time.Second)
	case finishPulling:
		nextExecEvtName = launchFromDisk
		nextExecEvtTime = event.Time.Add(1 * time.Second)
	case launchFromDisk:
		nextExecEvtName = finishLaunching
		nextExecEvtTime = event.Time.Add(10 * time.Second)
	case launchFromPageCache:
		nextExecEvtName = finishLaunching
		nextExecEvtTime = event.Time.Add(100 * time.Millisecond)
	case finishLaunching:
		nextReplicaEvtName = finishLaunchingReplica
		nextReplicaEvtTime = event.Time.Add(10 * time.Millisecond)
	case killProcess:
		nextReplicaEvtName = finishTerminatingReplica
		nextReplicaEvtTime = event.Time.Add(10 * time.Millisecond)
	}

	if event.EventName != killProcess && event.EventName != finishLaunching {
		execEvt := &simulator.Event{
			EventName:   nextExecEvtName,
			Time:        nextExecEvtTime,
			Subject:     e,
		}

		e.env.Schedule(execEvt)
	}

	if nextReplicaEvtName != "" {
		for _, r := range e.replicas {
			replicaEvt := &simulator.Event{
				EventName:   nextReplicaEvtName,
				Time:        nextReplicaEvtTime,
				Subject:     r,
			}

			r.nextEvt = replicaEvt

			e.env.Schedule(replicaEvt)
		}
	}

	current := e.fsm.Current()
	err := e.fsm.Event(event.EventName)
	if err != nil {
		panic(err.Error())
	}

	return simulator.TransitionResult{FromState: current, ToState: e.fsm.Current()}
}

func (e *Executable) Run(env *simulator.Environment, startingAt time.Time) {
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
		Time:        startingAt.Add(time.Duration(r) * time.Millisecond),
		EventName:   kickoffEventName,
		Subject:     e,
	})
}

func (e *Executable) AddRevisionReplica(replica *RevisionReplica) {
	e.replicas = append(e.replicas, replica)
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
