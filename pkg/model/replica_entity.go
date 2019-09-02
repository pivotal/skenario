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

	"knative.dev/serving/pkg/autoscaler"

	"skenario/pkg/simulator"
)

type Replica interface {
	RequestsProcessing() RequestsProcessingStock
	Stat() autoscaler.Stat
}

type ReplicaEntity interface {
	simulator.Entity
	Replica
}

type replicaEntity struct {
	env                  simulator.Environment
	number               int
	requestsProcessing   RequestsProcessingStock
	requestsComplete     simulator.SinkStock
}

var replicaNum int

func (re *replicaEntity) RequestsProcessing() RequestsProcessingStock {
	return re.requestsProcessing
}

func (re *replicaEntity) Stat() autoscaler.Stat {
	atTime := re.env.CurrentMovementTime()
	count := float64(re.requestsProcessing.Count())
	avgCount := float64(re.requestsProcessing.RequestCount())
	stat := autoscaler.Stat{
		Time:                      &atTime,
		PodName:                   string(re.Name()),
		RequestCount:              count,
		AverageConcurrentRequests: avgCount,
	}

	return stat
}

func (re *replicaEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("replica-%d", re.number))
}

func (re *replicaEntity) Kind() simulator.EntityKind {
	return "Replica"
}

func NewReplicaEntity(env simulator.Environment, totalConcurrencyCapacity int64) ReplicaEntity {
	replicaNum++

	re := &replicaEntity{
		env:               env,
		number:            replicaNum,
	}

	re.requestsComplete = simulator.NewSinkStock(simulator.StockName(fmt.Sprintf("RequestsComplete [%d]", re.number)), "Request")
	re.requestsProcessing = NewRequestsProcessingStock(env, re.number, re.requestsComplete, totalConcurrencyCapacity)

	return re
}
