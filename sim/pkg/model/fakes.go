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
	"context"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"time"

	"skenario/pkg/simulator"
)

type FakeEnvironment struct {
	Movements          []simulator.Movement
	TheTime            time.Time
	TheHaltTime        time.Time
	TheCPUUtilizations []*simulator.CPUUtilization
	ThePluginPartition string
}

func (fe *FakeEnvironment) PluginPartition() string {
	return fe.ThePluginPartition
}

func (fe *FakeEnvironment) AddToSchedule(movement simulator.Movement) (added bool) {
	fe.Movements = append(fe.Movements, movement)
	return true
}

func (fe *FakeEnvironment) Run() (completed []simulator.CompletedMovement, ignored []simulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func (fe *FakeEnvironment) CurrentMovementTime() time.Time {
	return fe.TheTime
}

func (fe *FakeEnvironment) HaltTime() time.Time {
	return fe.TheHaltTime
}

func (fe *FakeEnvironment) Context() context.Context {
	return context.Background()
}

func (fe *FakeEnvironment) CPUUtilizations() []*simulator.CPUUtilization {
	return fe.TheCPUUtilizations
}

func (fe *FakeEnvironment) AppendCPUUtilization(cpu *simulator.CPUUtilization) {
	fe.TheCPUUtilizations = append(fe.TheCPUUtilizations, cpu)
}

func NewFakeEnvironment() *FakeEnvironment {
	return &FakeEnvironment{
		ThePluginPartition: "1",
	}
}

type FakeReplica struct {
	ActivateCalled                     bool
	DeactivateCalled                   bool
	RequestsProcessingCalled           bool
	StatCalled                         bool
	FakeReplicaNum                     int
	ProcessingStock                    RequestsProcessingStock
	totalCPUCapacityMillisPerSecond    float64
	occupiedCPUCapacityMillisPerSecond float64
}

func (*FakeReplica) Name() simulator.EntityName {
	return "Replica"
}

func (*FakeReplica) Kind() simulator.EntityKind {
	return "Replica"
}

func (fr *FakeReplica) Activate() {
	fr.ActivateCalled = true
}

func (fr *FakeReplica) Deactivate() {
	fr.DeactivateCalled = true
}

func (fr *FakeReplica) RequestsProcessing() RequestsProcessingStock {
	fr.RequestsProcessingCalled = true
	failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
	if fr.ProcessingStock == nil {
		return NewRequestsProcessingStock(NewFakeEnvironment(), fr.FakeReplicaNum, simulator.NewSinkStock("fake-requestsComplete", "Request"),
			&failedSink, &fr.totalCPUCapacityMillisPerSecond, &fr.occupiedCPUCapacityMillisPerSecond)
	} else {
		return fr.ProcessingStock
	}
}

func (fr *FakeReplica) Stats() []*proto.Stat {
	fr.StatCalled = true
	return make([]*proto.Stat, 0)
}

func (fr *FakeReplica) GetCPUCapacity() float64 {
	return fr.totalCPUCapacityMillisPerSecond
}

type FakeDispatcher struct {
	scaleTimes []int64
	stats      []*proto.Stat
	scaleTo    int32
}

func (fp *FakeDispatcher) Event(partition string, time int64, typ proto.EventType, object skplug.Object) error {
	return nil
}

func (fp *FakeDispatcher) Stat(partition string, stat []*proto.Stat) error {
	fp.stats = append(fp.stats, stat...)
	return nil
}

func (fp *FakeDispatcher) HorizontalRecommendation(partition string, time int64) (rec int32, err error) {
	fp.scaleTimes = append(fp.scaleTimes, time)
	return fp.scaleTo, nil
}

func (fp *FakeDispatcher) VerticalRecommendation(partition string, time int64) (rec []*proto.RecommendedPodResources, err error) {
	//TODO implement it after injecting vertical scaling logic into Skenario
	panic("unimplemented")
}

func NewFakeDispatcher() *FakeDispatcher {
	return &FakeDispatcher{
		scaleTimes: make([]int64, 0),
		stats:      make([]*proto.Stat, 0),
	}
}
