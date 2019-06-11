package model

import (
	"fmt"
	"skenario/pkg/simulator"
)

type CpuStock interface {
	simulator.ThroughStock
}

type cpuStock struct {
	finished simulator.SinkStock
	requests []*requestEntity
}

func (cs *cpuStock) Add(entity simulator.Entity) error {
	req, ok := entity.(*requestEntity)
	if !ok {
		return fmt.Errorf("cpu stock wants requestEntity. got %T", entity)
	}
	if len(cs.requests) == 0 {
		// Schedule the first CPU tick.
	}
	cs.requests = append(cs.requests, req)
	return nil
}

func (cs *cpuStock) Count() uint64 {
	return len(requests)
}

func (cs *cpuStock) EntitiesInStock() []*Entity {
	return cs.requests
}

func (cs *cpuStock) KindStocked() EntityKind {
	return "Requests"
}

func (cs *cpuStock) Name() StockName {
	return "CPU"
}

func (cs *cpuStock) Remove() Entity {
	cnt := len(cs.requests)
	if cnt == 0 {
		return nil
	}
	r := cs.requests[0]
	r.cpuSecondsConsumed += time.Milliseconds * 100
	cs.requests = cs.requests[1:]
	if len(cs.requests) != 0 {
		// Schedule the next CPU tick.
	}
	if r.cpuSecondsRequired <= r.cpuSecondsConsumed {
		// Send the request to the finished sink.
		// Return the next request in the queue.
	}
	return r
}
