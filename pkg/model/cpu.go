package model

import (
	"fmt"
	"skenario/pkg/simulator"
	"time"
)

var timeSlice = 100 * time.Millisecond

type CpuStock interface {
	simulator.ThroughStock
}

type cpuStock struct {
	env        simulator.Environment
	terminated simulator.SinkStock
	processes  []*requestEntity
}

func (cpu *cpuStock) scheduleInterrupt() {
	p := cpu.processes[0]
	// Schedule the minimum of remaining required cpu seconds and
	// round-robin time-slice.
	duration := p.cpuSecondsRequired - p.cpuSecondsConsumed
	if duration > timeSlice {
		duration = timeSlice
	}
	// Record CPU consumption and schedule interrupt.
	p.cpuSecondsConsumed += duration
	cpu.env.AddToSchedule(simulator.NewMovement(
		"process_interrupt",
		cpu.env.CurrentMovementTime().Add(duration),
		cpu,
		cpu,
	))
}

func (cpu *cpuStock) Add(entity simulator.Entity) error {
	req, ok := entity.(*requestEntity)
	if !ok {
		return fmt.Errorf("cpu stock wants requestEntity. got %T", entity)
	}
	if req.cpuSecondsRequired <= req.cpuSecondsConsumed {
		return cpu.terminated.Add(req)
	}
	cpu.processes = append(cpu.processes, req)
	if len(cpu.processes) == 1 {
		// Schedule the first CPU tick.
		cpu.scheduleInterrupt()
	}
	return nil
}

func (cpu *cpuStock) Count() uint64 {
	return uint64(len(cpu.processes))
}

func (cpu *cpuStock) EntitiesInStock() []*simulator.Entity {
	es := make([]*simulator.Entity, len(cpu.processes))
	for i := 0; i < len(cpu.processes); i++ {
		e := cpu.processes[i]
		es[i] = &e
	}
	return es
}

func (cpu *cpuStock) KindStocked() simulator.EntityKind {
	return "Processes"
}

func (cpu *cpuStock) Name() simulator.StockName {
	return "CPU"
}

func (cpu *cpuStock) Remove() Entity {
	if len(cpu.processes) == 0 {
		return nil
	}
	p := cpu.processes[0]
	cpu.processes = cpu.processes[1:]
	if len(cpu.processes) != 0 {
		// Schedule the next CPU tick.
		cpu.scheduleInterrupt()
	}
	return p
}

func NewCpuStock(env simulator.Env, requestSink simulator.SinkStock) {
	return &cpuStock{
		env:        env,
		terminated: requestSink,
		processes:  make([]*requestEntity, 0),
	}
}
