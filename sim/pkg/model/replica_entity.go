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
	"github.com/josephburnett/sk-plugin/pkg/skplug"

	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"skenario/pkg/simulator"
)

type Replica interface {
	Activate()
	Deactivate()
	RequestsProcessing() RequestsProcessingStock
	Stats() []*proto.Stat
	GetCPUCapacity() float64
}

type ReplicaEntity interface {
	simulator.Entity
	Replica
}

type replicaEntity struct {
	env                                simulator.Environment
	number                             int
	requestsProcessing                 RequestsProcessingStock
	requestsComplete                   simulator.SinkStock
	requestsFailed                     simulator.SinkStock
	numRequestsSinceStat               int32
	totalCPUCapacityMillisPerSecond    float64
	occupiedCPUCapacityMillisPerSecond float64
}

var replicaNum int

func (re *replicaEntity) Activate() {
	now := re.env.CurrentMovementTime().UnixNano()
	err := re.env.PluginDispatcher().Event(now, proto.EventType_CREATE, &skplug.Pod{
		Name: string(re.Name()),
		// TODO: enumerate states in proto.
		State:          "active",
		LastTransition: now,
		CpuRequest:     int32(re.GetCPUCapacity()),
	})
	if err != nil {
		panic(err)
	}
}

func (re *replicaEntity) Deactivate() {
	now := re.env.CurrentMovementTime().UnixNano()
	err := re.env.PluginDispatcher().Event(now, proto.EventType_DELETE, &skplug.Pod{
		Name: string(re.Name()),
	})
	if err != nil {
		panic(err)
	}
}

func (re *replicaEntity) RequestsProcessing() RequestsProcessingStock {
	return re.requestsProcessing
}

func (re *replicaEntity) Stats() []*proto.Stat {
	atTime := re.env.CurrentMovementTime()
	stats := make([]*proto.Stat, 0)

	stats = append(stats, &proto.Stat{
		Time:    atTime.UnixNano(),
		PodName: string(re.Name()),
		Type:    proto.MetricType_CONCURRENT_REQUESTS_MILLIS,
		Value:   int32(re.requestsProcessing.Count() * 1000),
	})
	cpuUsage := int32(re.occupiedCPUCapacityMillisPerSecond)
	stats = append(stats, &proto.Stat{
		Time:    atTime.UnixNano(),
		PodName: string(re.Name()),
		Type:    proto.MetricType_CPU_MILLIS,
		Value:   cpuUsage,
	})

	re.numRequestsSinceStat = 0
	// TODO: report request count

	return stats
}

func (re *replicaEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("replica-%d", re.number))
}

func (re *replicaEntity) Kind() simulator.EntityKind {
	return "Replica"
}

func (re *replicaEntity) GetCPUCapacity() float64 {
	return re.totalCPUCapacityMillisPerSecond
}

func NewReplicaEntity(env simulator.Environment, failedSink *simulator.SinkStock) ReplicaEntity {
	replicaNum++

	re := &replicaEntity{
		env:                                env,
		number:                             replicaNum,
		totalCPUCapacityMillisPerSecond:    100,
		occupiedCPUCapacityMillisPerSecond: 0,
	}

	re.requestsComplete = simulator.NewSinkStock(simulator.StockName(fmt.Sprintf("RequestsComplete [%d]", re.number)), "Request")
	re.requestsProcessing = NewRequestsProcessingStock(env, re.number, re.requestsComplete, failedSink, &re.totalCPUCapacityMillisPerSecond, &re.occupiedCPUCapacityMillisPerSecond)

	return re
}
