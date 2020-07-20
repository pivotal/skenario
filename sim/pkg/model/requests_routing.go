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
	"time"

	"skenario/pkg/simulator"
)

type RequestsRoutingStock interface {
	simulator.ThroughStock
}

type requestsRoutingStock struct {
	env            simulator.Environment
	delegate       simulator.ThroughStock
	replicas       ReplicasActiveStock
	requestsFailed simulator.SinkStock
	countRequests  int
}

func (rbs *requestsRoutingStock) Name() simulator.StockName {
	return rbs.delegate.Name()
}

func (rbs *requestsRoutingStock) KindStocked() simulator.EntityKind {
	return rbs.delegate.KindStocked()
}

func (rbs *requestsRoutingStock) Count() uint64 {
	return rbs.delegate.Count()
}

func (rbs *requestsRoutingStock) EntitiesInStock() map[simulator.Entity]bool {
	return rbs.delegate.EntitiesInStock()
}
func (rbs *requestsRoutingStock) GetEntityByNumber(number int) simulator.Entity {
	return rbs.delegate.GetEntityByNumber(number)
}

func (rbs *requestsRoutingStock) Remove(entity *simulator.Entity) simulator.Entity {
	return rbs.delegate.Remove(entity)
}

func (rbs *requestsRoutingStock) Add(entity simulator.Entity) error {
	addResult := rbs.delegate.Add(entity)

	rbs.countRequests++

	countReplicas := rbs.replicas.Count()
	if countReplicas > 0 {
		replica := rbs.replicas.GetEntityByNumber(int(uint64(rbs.countRequests) % countReplicas)).(ReplicaEntity)

		rbs.env.AddToSchedule(simulator.NewMovement(
			"send_to_replica",
			rbs.env.CurrentMovementTime().Add(1*time.Nanosecond),
			rbs,
			replica.RequestsProcessing(),
			&entity,
		))
	} else {
		rbs.env.AddToSchedule(simulator.NewMovement(
			"request_failed",
			rbs.env.CurrentMovementTime().Add(1*time.Nanosecond),
			rbs,
			rbs.requestsFailed,
			&entity,
		))
	}

	return addResult
}

func NewRequestsRoutingStock(env simulator.Environment, replicas ReplicasActiveStock, requestsFailed simulator.SinkStock) RequestsRoutingStock {
	return &requestsRoutingStock{
		env:            env,
		delegate:       simulator.NewThroughStock("RequestsRouting", "Request"),
		replicas:       replicas,
		requestsFailed: requestsFailed,
		countRequests:  0,
	}
}
