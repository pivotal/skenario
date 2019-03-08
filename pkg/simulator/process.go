package simulator

import (
	"fmt"
	"math/rand"
	"time"
)

type Process struct {
	Name        string
	Environment *Environment
}

func (p *Process) Advance(t time.Time, description string) {
	fmt.Printf("[%d]: %s", t.UnixNano(), description)
	if rand.Intn(100) > 5 {
		nextTime := t.Add(100 * time.Millisecond)
		evt := &Event{
			Time:        nextTime,
			Description: fmt.Sprintf("Scheduled event for %d\n", nextTime.UnixNano()),
			AdvanceFunc: p.Advance,
		}

		p.Environment.Schedule(evt)
	}

	return
}

func (p *Process) Run() {
	evt := &Event{
		Time:        time.Now().Add(111 * time.Millisecond),
		Description: "process.Run()\n",
		AdvanceFunc: p.Advance,
	}
	p.Environment.Schedule(evt)
}
