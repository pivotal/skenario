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
	"math"
	"math/rand"
	"time"

	"skenario/pkg/simulator"
)

type RequestsProcessingStock interface {
	simulator.ThroughStock
	RequestCount() int32
}

type requestsProcessingStock struct {
	env                       simulator.Environment
	delegate                  simulator.ThroughStock
	replicaNumber             int
	requestsComplete          simulator.SinkStock
	requestsFailed            simulator.SinkStock
	numRequestsSinceLast      int32
	totalCPUCapacityMillis    *int
	occupiedCPUCapacityMillis *int
}

func (rps *requestsProcessingStock) Name() simulator.StockName {
	name := fmt.Sprintf("%s [%d]", rps.delegate.Name(), rps.replicaNumber)
	return simulator.StockName(name)
}

func (rps *requestsProcessingStock) KindStocked() simulator.EntityKind {
	return rps.delegate.KindStocked()
}

func (rps *requestsProcessingStock) Count() uint64 {
	return rps.delegate.Count()
}

func (rps *requestsProcessingStock) EntitiesInStock() []*simulator.Entity {
	return rps.delegate.EntitiesInStock()
}

func (rps *requestsProcessingStock) Remove() simulator.Entity {
	request := rps.delegate.Remove().(*requestEntity)
	*rps.occupiedCPUCapacityMillis -= *request.utilizationForRequest
	return request
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	rps.numRequestsSinceLast++
	request := *entity.(*requestEntity)

	freeCPUCapacityMillis := *rps.totalCPUCapacityMillis - *rps.occupiedCPUCapacityMillis
	if freeCPUCapacityMillis == 0 {
		freeCPUCapacityMillis = 1
	}
	cpuTime := request.requestConfig.CPUTimeMillis * 1000 / freeCPUCapacityMillis
	processingTime := cpuTime + request.requestConfig.IOTimeMillis
	utilizationForRequest := cpuTime * freeCPUCapacityMillis / processingTime
	*request.utilizationForRequest = utilizationForRequest
	*rps.occupiedCPUCapacityMillis += utilizationForRequest

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	//calculate currentUtilization in percentage
	currentUtilization := *rps.occupiedCPUCapacityMillis * 100 / *rps.totalCPUCapacityMillis
	totalTime := calculateTime(currentUtilization, time.Duration(processingTime)*time.Millisecond, rng)

	if totalTime > request.requestConfig.Timeout {
		*rps.occupiedCPUCapacityMillis -= utilizationForRequest

		rps.env.AddToSchedule(simulator.NewMovement(
			"request_failed",
			rps.env.CurrentMovementTime().Add(1*time.Nanosecond),
			rps,
			rps.requestsFailed,
		))
	} else {
		rps.env.AddToSchedule(simulator.NewMovement(
			"complete_request",
			rps.env.CurrentMovementTime().Add(totalTime),
			rps,
			rps.requestsComplete,
		))
	}

	return rps.delegate.Add(entity)
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestComplete simulator.SinkStock,
	requestFailed simulator.SinkStock, totalCPUCapacityMillis *int, occupiedCPUCapacityMillis *int) RequestsProcessingStock {
	return &requestsProcessingStock{
		env:                       env,
		delegate:                  simulator.NewThroughStock("RequestsProcessing", "Request"),
		replicaNumber:             replicaNumber,
		requestsComplete:          requestComplete,
		requestsFailed:            requestFailed,
		occupiedCPUCapacityMillis: occupiedCPUCapacityMillis,
		totalCPUCapacityMillis:    totalCPUCapacityMillis,
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

func calculateTime(currentUtilization int, baseServiceTime time.Duration, rng *rand.Rand) time.Duration {
	fractionUtilised := saturateClamp(float64(currentUtilization) / float64(100))
	delayTime := 1 + sakasegawaApproximation(fractionUtilised, float64(100), baseServiceTime)

	delayRand := rng.Int63n(int64(delayTime))
	totalTime := baseServiceTime + time.Duration(delayRand)

	return totalTime
}
