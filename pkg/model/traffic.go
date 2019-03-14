package model

import (
	"math/rand"
	"time"

	"knative-simulator/pkg/simulator"
)

type Traffic struct {
	env       *simulator.Environment
	replica   *RevisionReplica
	beginTime time.Time
	endTime   time.Time
}

func NewTraffic(env *simulator.Environment, replica *RevisionReplica, begin time.Time, runFor time.Duration) *Traffic {
	return &Traffic{
		env:       env,
		replica:   replica,
		beginTime: begin,
		endTime:   begin.Add(runFor),
	}
}

func (tr *Traffic) Run() {
	t := tr.beginTime

	for {
		r := rand.Int63n(5000)
		t = t.Add(time.Duration(r) * time.Millisecond)

		if t.After(tr.endTime) {
			return
		}

		tr.env.Schedule(&simulator.Event{
			Time:        t,
			EventName:   receiveRequest,
			AdvanceFunc: tr.replica.Advance,
		})
	}
}
