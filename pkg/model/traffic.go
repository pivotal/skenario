package model

import (
	"math/rand"
	"time"

	"knative-simulator/pkg/simulator"
)

type Traffic struct {
	env       *simulator.Environment
	replica   *RevisionReplica
	buffer    *KBuffer
	beginTime time.Time
	endTime   time.Time
}

func NewTraffic(env *simulator.Environment, buffer *KBuffer, replica *RevisionReplica, begin time.Time, runFor time.Duration) *Traffic {
	return &Traffic{
		env:       env,
		replica:   replica,
		buffer:    buffer,
		beginTime: begin,
		endTime:   begin.Add(runFor),
	}
}

func (tr *Traffic) Run() {
	t := tr.beginTime

	for {
		r := rand.Int63n(10000)
		t = t.Add(time.Duration(r) * time.Millisecond)

		if t.After(tr.endTime) {
			return
		}

		req := NewRequest(tr.env, tr.buffer, tr.replica, t)
		req.Run()
	}
}
