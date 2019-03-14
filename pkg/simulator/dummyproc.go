package simulator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"
)

type dummyProc struct {
	name     string
	env      *Environment
	fsm      *fsm.FSM
	evtCount map[string]int
}

func NewDummyProc(name, initialState string) *dummyProc {
	dp := &dummyProc{
		name: name,
		evtCount: make(map[string]int, 0),
	}

	dp.fsm = fsm.NewFSM(
		initialState,
		fsm.Events{
			{Name: "foo->bar", Src: []string{"FOO"}, Dst: "BAR"},
			{Name: "bar->foo", Src: []string{"BAR"}, Dst: "FOO"},
		},
		fsm.Callbacks{},
	)

	return dp
}

func (dp *dummyProc) Advance(t time.Time, eventName string) (identifier, fromState, toState, note string) {
	r := rand.Int63n(int64(time.Second * 30))
	add := time.Duration(r) * time.Nanosecond
	nextTime := t.Add(add)
	dp.evtCount[eventName]++

	dp.env.Schedule(&Event{
		Time:        nextTime,
		EventName:   dp.fsm.AvailableTransitions()[0],
		AdvanceFunc: dp.Advance,
	})

	return dp.name, "ignore", "ignore", fmt.Sprintf("# %d", dp.evtCount[eventName])
}

func (dp *dummyProc) Run(env *Environment) {
	dp.env = env

	r := rand.Intn(10000)
	t := time.Unix(0, int64(r)).UTC()

	evt := &Event{
		Time:        t,
		EventName:   "foo->bar",
		AdvanceFunc: dp.Advance,
	}

	dp.env.Schedule(evt)
}
