package model

import (
	"fmt"
	"math/rand"
	"time"

	"knative-simulator/pkg/simulator"
)

type Traffic struct {
	env       *simulator.Environment
	endpoints *ReplicaEndpoints
	buffer    *KBuffer
	beginTime time.Time
	endTime   time.Time
}

func NewTraffic(env *simulator.Environment, buffer *KBuffer, endpoints *ReplicaEndpoints, begin time.Time, runFor time.Duration) *Traffic {
	return &Traffic{
		env:       env,
		endpoints: endpoints,
		buffer:    buffer,
		beginTime: begin,
		endTime:   begin.Add(runFor),
	}
}

func (tr *Traffic) Identity() simulator.ProcessIdentity {
	return "Traffic"
}

func (tr *Traffic) AddStock(item simulator.Stockable) {
	panic("not implemented")
}

func (tr *Traffic) RemoveStock(item simulator.Stockable) {
	// do nothing, this is to match the Stock type
}

func (tr *Traffic) Run() {
	t := tr.beginTime

	for {
		r := rand.Int63n(100000)
		t = t.Add(time.Duration(r) * time.Millisecond)

		if t.After(tr.endTime) {
			return
		}

		req := NewRequest(tr.env, tr.buffer, t)
		tr.env.Schedule(simulator.NewMovementEvent(
			simulator.EventName(fmt.Sprintf("move-%s", req.name)),
			t,
			req,
			tr,
			tr.buffer,
		))
	}
}
