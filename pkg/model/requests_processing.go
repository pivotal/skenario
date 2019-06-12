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
	"skenario/pkg/simulator"
)

type RequestsProcessingStock interface {
	simulator.ThroughStock
	RequestCount() int32
}

type requestsProcessingStock struct {
	env                  simulator.Environment
	replicaNumber        int
	cpu                  *cpuStock
	numRequestsSinceLast int32
}

func (rps *requestsProcessingStock) Name() simulator.StockName {
	return simulator.StockName(fmt.Sprintf("RequestsProcessing [%d]", rps.replicaNumber))
}

func (rps *requestsProcessingStock) KindStocked() simulator.EntityKind {
	return "Request"
}

func (rps *requestsProcessingStock) Count() uint64 {
	return rps.cpu.Count()
}

func (rps *requestsProcessingStock) EntitiesInStock() []*simulator.Entity {
	return rps.cpu.EntitiesInStock()
}

func (rps *requestsProcessingStock) Remove() simulator.Entity {
	return rps.cpu.Remove()
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	rps.numRequestsSinceLast++
	// TODO: non-cpu-bound requests
	return rps.cpu.Add(entity)
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestSink simulator.SinkStock, replicaMaxRPSCapacity int64) RequestsProcessingStock {
	// TODO: respect replicaMaxRPSCapacity
	return &requestsProcessingStock{
		env:           env,
		replicaNumber: replicaNumber,
		cpu:           NewCpuStock(env, requestSink),
	}
}
