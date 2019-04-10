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

type requestsProcessingStock struct {
	env                  simulator.Environment
	delegate             simulator.ThroughStock
	replicaNumber        int
	requestsComplete     simulator.SinkStock
	numRequestsSinceLast int32
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
	return rps.delegate.Remove()
}

func (rps *requestsProcessingStock) Add(entity simulator.Entity) error {
	rps.numRequestsSinceLast++

	rps.env.AddToSchedule(simulator.NewMovement(
		"complete_request",
		rps.env.CurrentMovementTime().Add(1*time.Second),
		rps,
		rps.requestsComplete,
	))
	return rps.delegate.Add(entity)
}

func (rps *requestsProcessingStock) RequestCount() int32 {
	rc := rps.numRequestsSinceLast
	rps.numRequestsSinceLast = 0
	return rc
}

func NewRequestsProcessingStock(env simulator.Environment, replicaNumber int, requestSink simulator.SinkStock) RequestsProcessingStock {
	return &requestsProcessingStock{
		env:              env,
		delegate:         simulator.NewThroughStock("RequestsProcessing", "Request"),
		replicaNumber:    replicaNumber,
		requestsComplete: requestSink,
	}
}
