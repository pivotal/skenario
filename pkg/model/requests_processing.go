/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package model

import (
	"fmt"
	"time"

	"skenario/pkg/simulator"
)

type RequestsProcessingStock interface {
	simulator.ThroughStock
	RequestCount() int32
}

type cpuUsage struct {
	window           time.Duration
	activeTimeSlices [][2]time.Time
}

func NewCpuUsage(window time.Duration) *cpuUsage {
	return &cpuUsage{
		window:           window,
		activeTimeSlices: make([][2]time.Time, 0),
	}
}

func (c *cpuUsage) active(from, to time.Time) {
	c.activeTimeSlices = append(c.activeTimeSlices, [2]time.Time{from, to})
}

func (c *cpuUsage) trim(now time.Time) {
	trimmed := make([][2]time.Time, 0)
	for _, t := range c.activeTimeSlices {
		if t[1].Before(now.Add(-c.window)) {
			continue
		}
		if t[0].Before(now.Add(-c.window)) {
			t[0] = now.Add(-c.window)
		}
		trimmed = append(trimmed, t)
	}
	c.activeTimeSlices = trimmed
}

func (c *cpuUsage) usage(now time.Time) float64 {
	c.trim(now)
	var activeNanos int64
	for _, a := range c.activeTimeSlices {
		activeNanos += a[1].Sub(a[0]).Nanoseconds()
	}
	return float64(activeNanos) / float64(c.window.Nanoseconds())
}

type requestsProcessingStock struct {
	env                                simulator.Environment
	delegate                           simulator.ThroughStock
	replicaNumber                      int
	requestsComplete                   simulator.SinkStock
	requestsFailed                     *simulator.SinkStock
	requestsExhausted                  simulator.ThroughStock
	numRequestsSinceLast               int32
	totalCPUCapacityMillisPerSecond    *float64
	occupiedCPUCapacityMillisPerSecond *float64

	// Internal process accounting.
	processesActive     simulator.ThroughStock
	processesOnCpu      simulator.ThroughStock
	processesTerminated simulator.ThroughStock
	cpuUsage            *cpuUsage
}

func (rps *requestsProcessingStock) Name() simulator.StockName {
	name := fmt.Sprintf("%s [%d]", rps.processesActive.Name(), rps.replicaNumber)
	return simulator.StockName(name)
}

func (rps *requestsProcessingStock) KindStocked() simulator.EntityKind {
	return rps.processesActive.KindStocked()
}

func (rps *requestsProcessingStock) Count() uint64 {
	return rps.processesActive.Count() + rps.processesOnCpu.Count()
}

func (rps *requestsProcessingStock) EntitiesInStock() []*simulator.Entity {
	entities := make([]*simulator.Entity, rps.processesActive.Count()+rps.processesOnCpu.Count())
	entities = append(entities, rps.processesActive.EntitiesInStock()...)
	entities = append(entities, rps.processesOnCpu.EntitiesInStock()...)
	return entities
}

func (rps *requestsProcessingStock) Remove() simulator.Entity {
	request := rps.delegate.Remove().(*requestEntity)
	*rps.occupiedCPUCapacityMillisPerSecond -= *request.utilizationForRequestMillisPerSecond
	return request

	//if rps.processesTerminated.Count() > 0 {
	//	return rps.processesTerminated.Remove()
	//}
	//return nil
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	var totalTime time.Duration

	// TODO: this isn't correct anymore because it's used for interrupts.
	//rps.numRequestsSinceLast++

	request, ok := *entity.(*requestEntity)
	if !ok {
		return fmt.Errorf("requests processing stock only supports request entities. got %T", entity)
	}

	isRequestSuccessful := true

	rps.calculateCPUUtilizationForRequest(request, &totalTime, &isRequestSuccessful)

	if isRequestSuccessful {
		rps.env.AddToSchedule(simulator.NewMovement(
			"complete_request",
			rps.env.CurrentMovementTime().Add(totalTime),
			rps,
			rps.requestsComplete,
		))
	} else {
		rps.env.AddToSchedule(simulator.NewMovement(
			"request_failed",
			rps.env.CurrentMovementTime().Add(request.requestConfig.Timeout),
			rps,
			*rps.requestsFailed,
		))
	}

	return rps.delegate.Add(entity)

	//now := rps.env.CurrentMovementTime()
	//if req.startTime == nil {
	//	req.startTime = &now
	//}
	//
	//// Enqueue or complete the request.
	//if req.cpuSecondsRemaining() > 0 {
	//	err := rps.processesActive.Add(entity)
	//	if err != nil {
	//		return err
	//	}
	//} else {
	//	err := rps.processesTerminated.Add(entity)
	//	if err != nil {
	//		return err
	//	}
	//	rps.env.AddToSchedule(simulator.NewMovement(
	//		"complete_request",
	//		now.Add(time.Nanosecond),
	//		rps,
	//		rps.requestsComplete,
	//	))
	//	// latency := now.Sub(*req.startTime)
	//	// log.Printf("latecy: %v\n", latency)
	//}
	//
	//// Fill the CPU and schedule an interrupt.
	//if rps.processesOnCpu.Count() == 0 && rps.processesActive.Count() > 0 {
	//	req := rps.processesActive.Remove().(*requestEntity)
	//	interruptAfter := req.cpuSecondsRemaining()
	//	if interruptAfter > 200*time.Millisecond {
	//		interruptAfter = 200 * time.Millisecond
	//	}
	//	interruptAt := now.Add(interruptAfter)
	//	req.cpuSecondsConsumed += interruptAfter
	//	rps.env.AddToSchedule(simulator.NewMovement(
	//		"interrupt_process",
	//		interruptAt,
	//		rps.processesOnCpu,
	//		rps,
	//	))
	//	rps.cpuUsage.active(now, interruptAt)
	//	return rps.processesOnCpu.Add(entity)
	//}
	//return nil
}

func (rps *requestsProcessingStock) calculateCPUUtilizationForRequest(request requestEntity, totalTime *time.Duration, isRequestSuccessful *bool) {
	//step 1 calculate free cpu capacity
	freeCPUCapacityMillisPerSecond := *rps.totalCPUCapacityMillisPerSecond - *rps.occupiedCPUCapacityMillisPerSecond
	const eps = 0.001
	if freeCPUCapacityMillisPerSecond > eps {
		//step 2 Calculate how many cpu time we need to process this request, need to multiply by 1000
		//to get cpuTimeMillis in milliseconds
		cpuTimeMillis := float64(request.requestConfig.CPUTimeMillis) * 1000 / freeCPUCapacityMillisPerSecond

		//step 3 Calculate how many time we need to process this request taking into account io time
		processingTimeMillis := cpuTimeMillis + float64(request.requestConfig.IOTimeMillis)

		//step 4 Calculate average cpu load for the request that is utilization for the request
		utilizationForRequestMillisPerSecond := cpuTimeMillis * freeCPUCapacityMillisPerSecond / processingTimeMillis

		*request.utilizationForRequestMillisPerSecond = utilizationForRequestMillisPerSecond

		//step 5 Add  this utilization to occupied cpu capacity, we'll subtract it Remove() method
		*rps.occupiedCPUCapacityMillisPerSecond += utilizationForRequestMillisPerSecond

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		//step 6 Calculate currentUtilization in percentage
		currentUtilization := *rps.occupiedCPUCapacityMillisPerSecond * 100 / *rps.totalCPUCapacityMillisPerSecond

		//step 7 Calculate delay by sakasegawaApproximation which plus processing time forms total time for processing a request
		*totalTime = calculateTime(currentUtilization, time.Duration(processingTimeMillis)*time.Millisecond, rng)

		*isRequestSuccessful = *totalTime <= request.requestConfig.Timeout
	} else {
		*isRequestSuccessful = false
	}
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestComplete simulator.SinkStock,
	requestFailed *simulator.SinkStock, totalCPUCapacityMillisPerSecond *float64, occupiedCPUCapacityMillisPerSecond *float64) RequestsProcessingStock {
	return &requestsProcessingStock{
		env:                                env,
		processesActive:                    simulator.NewThroughStock("RequestsProcessing", "Request"),
		processesOnCpu:                     simulator.NewThroughStock("RequestsProcessing", "Request"),
		processesTerminated:                simulator.NewThroughStock("RequestsProcessing", "Request"),
		delegate:                           simulator.NewThroughStock("RequestsProcessing", "Request"),
		replicaNumber:                      replicaNumber,
		requestsComplete:                   requestComplete,
		requestsFailed:                     requestFailed,
		requestsExhausted:                  simulator.NewThroughStock("RequestsProcessing", "Request"),
		occupiedCPUCapacityMillisPerSecond: occupiedCPUCapacityMillisPerSecond,
		totalCPUCapacityMillisPerSecond:    totalCPUCapacityMillisPerSecond,
		cpuUsage:              NewCpuUsage(15 * time.Second),
	}
}

func saturateClamp(fractionUtilised float64) float64 {
	if fractionUtilised > 0.96 {
		return 0.96
	} else if fractionUtilised > 0.0 {
		return fractionUtilised
	}

	return 0.01
}

// returns Sakasegawa's approximation for expected queueing time for an M/M/m queue
func sakasegawaApproximation(fractionUtilised, totalCPUCapacity float64, baseServiceTime time.Duration) time.Duration {
	powerTerm := math.Sqrt(2*(totalCPUCapacity+1)) - 1
	utilizationTerm := math.Pow(fractionUtilised, powerTerm) / (totalCPUCapacity * (1 - fractionUtilised))

	expected := time.Duration(utilizationTerm * float64(baseServiceTime))

	return expected
}

func calculateTime(currentUtilization float64, baseServiceTime time.Duration, rng *rand.Rand) time.Duration {
	fractionUtilised := saturateClamp(currentUtilization / 100)
	delayTime := 1 + sakasegawaApproximation(fractionUtilised, float64(100), baseServiceTime)

	delayRand := rng.Int63n(int64(delayTime))
	totalTime := baseServiceTime + time.Duration(delayRand)

	return totalTime
}
