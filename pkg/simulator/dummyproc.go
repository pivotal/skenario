package simulator

import (
	"fmt"
	"math/rand"
	"time"
)

type dummyProc struct {
	name  string
	env   *Environment
	count int
}

func NewDummyProc(name string) Process {
	return &dummyProc{
		name:  name,
		count: 0,
	}
}

func (p *dummyProc) Advance(t time.Time, description string) {
	fmt.Printf("[%d] %s\n", t.UnixNano(), description)
	r := rand.Intn(1000000)
	add := time.Duration(r) * time.Nanosecond
	nextTime := t.Add(add)

	fmt.Printf("[%d] Scheduled event for %d\n", t.UnixNano(), nextTime.UnixNano())

	p.count += 1
	evt := &Event{
		Time:        nextTime,
		Description: fmt.Sprintf("%s %d", p.name, p.count),
		AdvanceFunc: p.Advance,
	}

	p.env.Schedule(evt)

	return
}

func (p *dummyProc) Run(env *Environment) {
	p.env = env

	r := rand.Intn(10000)
	t := time.Unix(0, int64(r)).UTC()

	evt := &Event{
		Time:        t,
		Description: fmt.Sprintf("process.Run() %s", p.name),
		AdvanceFunc: p.Advance,
	}

	p.env.Schedule(evt)
}
