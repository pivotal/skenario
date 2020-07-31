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
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"math"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/simulator"
)

func TestAutoscalerTicktock(t *testing.T) {
	spec.Run(t, "Ticktock stock", testAutoscalerTicktock, spec.Report(report.Terminal{}))
}

func testAutoscalerTicktock(t *testing.T, describe spec.G, it spec.S) {
	var subject AutoscalerTicktockStock
	var rawSubject *autoscalerTicktockStock
	var envFake *FakeEnvironment
	var replicasConfig ReplicasConfig
	var cluster ClusterModel

	it.Before(func() {
		envFake = NewFakeEnvironment()
		envFake.TheTime = time.Unix(0, 0)

		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		cluster = NewCluster(envFake, ClusterConfig{}, replicasConfig)
		subject = NewAutoscalerTicktockStock(envFake, simulator.NewEntity("Autoscaler", "HPAAutoscaler"), cluster)
		rawSubject = subject.(*autoscalerTicktockStock)
	})

	describe("NewAutoscalerTicktockStock()", func() {
		it("sets the entity", func() {
			assert.Equal(t, simulator.EntityName("Autoscaler"), rawSubject.autoscalerEntity.Name())
			assert.Equal(t, simulator.EntityKind("HPAAutoscaler"), rawSubject.autoscalerEntity.Kind())
		})
	})

	describe("KindStocked()", func() {
		it("accepts HPA Autoscalers", func() {
			assert.Equal(t, subject.KindStocked(), simulator.EntityKind("HPAAutoscaler"))
		})
	})

	describe("Count()", func() {
		it("always has 1 entity stocked", func() {
			assert.Equal(t, subject.Count(), uint64(1))

			ent := subject.Remove(nil)
			err := subject.Add(ent)
			assert.NoError(t, err)
			err = subject.Add(ent)
			assert.NoError(t, err)

			assert.Equal(t, subject.Count(), uint64(1))

			subject.Remove(nil)
			subject.Remove(nil)
			subject.Remove(nil)
			assert.Equal(t, subject.Count(), uint64(1))
		})
	})

	describe("Remove()", func() {
		it("gives back the one HPAAutoscaler", func() {
			assert.Equal(t, subject.Remove(nil), subject.Remove(nil))
		})
	})

	describe("Add()", func() {
		describe("ensuring consistency", func() {
			var differentEntity simulator.Entity

			it.Before(func() {
				differentEntity = simulator.NewEntity("Different!", "HPAAutoscaler")
			})

			it("returns error if the Added entity does not equal the existing entity", func() {
				err := subject.Add(differentEntity)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "different from the entity given at creation time")
			})
		})

		describe("driving the HPA autoscaler", func() {
			describe("controlling time", func() {
				it.Before(func() {
					ent := subject.Remove(nil)
					err := subject.Add(ent)
					assert.NoError(t, err)
				})

				it("triggers the autoscaler calculation with the current time", func() {
					assert.Equal(t, time.Unix(0, 0).UnixNano(), envFake.PluginDispatcher().(*FakeHpaPluginPartition).scaleTimes[0])
				})
			})

			describe("updating statistics", func() {
				var rawCluster *clusterModel

				it.Before(func() {
					rawCluster = cluster.(*clusterModel)
					failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
					newReplica := NewReplicaEntity(envFake, &failedSink)
					err := rawCluster.replicasActive.Add(newReplica)
					assert.NoError(t, err)

					ent := subject.Remove(nil)
					err = subject.Add(ent)
					assert.NoError(t, err)
				})

				it("delegates statistics updating to ClusterModel", func() {
					stats := envFake.ThePluginDispatcher.(*FakeHpaPluginPartition).stats
					assert.Len(t, stats, 3)
					assert.Equal(t, stats[0].Type, proto.MetricType_CONCURRENT_REQUESTS_MILLIS)
					assert.Equal(t, stats[1].Type, proto.MetricType_CONCURRENT_REQUESTS_MILLIS)
					assert.Equal(t, stats[2].Type, proto.MetricType_CPU_MILLIS)
				})
			})

			describe("the autoscaler was able to make a recommendation", func() {
				describe("to scale up", func() {
					it.Before(func() {
						envFake.ThePluginDispatcher.(*FakeHpaPluginPartition).scaleTo = 8
						err := cluster.Desired().Add(simulator.NewEntity("desired-1", "Desired"))
						assert.NoError(t, err)

						ent := subject.Remove(nil)
						err = subject.Add(ent)
						assert.NoError(t, err)
					})

					it("schedules movements into the ReplicasDesired stock", func() {
						assert.Equal(t, simulator.MovementKind("increase_desired"), envFake.Movements[8].Kind())
						assert.Equal(t, simulator.StockName("DesiredSource"), envFake.Movements[8].From().Name())
						assert.Equal(t, simulator.StockName("ReplicasDesired"), envFake.Movements[8].To().Name())
					})
				})

				describe("to scale down", func() {
					it.Before(func() {
						err := cluster.Desired().Add(simulator.NewEntity("desired-1", "Desired"))
						assert.NoError(t, err)
						err = cluster.Desired().Add(simulator.NewEntity("desired-1", "Desired"))
						assert.NoError(t, err)

						envFake.ThePluginDispatcher.(*FakeHpaPluginPartition).scaleTo = 1
						ent := subject.Remove(nil)
						err = subject.Add(ent)
						assert.NoError(t, err)
					})

					it("schedules movements out of the ReplicasDesired stock", func() {
						assert.Equal(t, simulator.MovementKind("reduce_desired"), envFake.Movements[4].Kind())
						assert.Equal(t, simulator.StockName("ReplicasDesired"), envFake.Movements[4].From().Name())
						assert.Equal(t, simulator.StockName("DesiredSink"), envFake.Movements[4].To().Name())
					})
				})
			})
			it("cpu utilization list is empty in environment", func() {
				assert.Equal(t, 0, len(envFake.TheCPUUtilizations))
			})

			describe("update cpu utilization list", func() {
				it.Before(func() {
					rawCluster := cluster.(*clusterModel)
					failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
					newReplica1 := NewReplicaEntity(envFake, &failedSink)
					newReplica1.(*replicaEntity).occupiedCPUCapacityMillisPerSecond = 50
					newReplica1.(*replicaEntity).totalCPUCapacityMillisPerSecond = 100
					err := rawCluster.replicasActive.Add(newReplica1)
					assert.NoError(t, err)

					newReplica2 := NewReplicaEntity(envFake, &failedSink)
					newReplica2.(*replicaEntity).occupiedCPUCapacityMillisPerSecond = 0
					newReplica2.(*replicaEntity).totalCPUCapacityMillisPerSecond = 100
					err = rawCluster.replicasActive.Add(newReplica2)
					assert.NoError(t, err)

					ent := subject.Remove(nil)
					err = subject.Add(ent)
					assert.NoError(t, err)

				})
				it("cpu utilization list is not empty in environment", func() {
					assert.NotEqual(t, 0, len(envFake.TheCPUUtilizations))
				})
				it("calculated average cpu utilization value is 25%", func() {
					index := len(envFake.TheCPUUtilizations) - 1
					assert.Less(t, math.Abs(envFake.TheCPUUtilizations[index].CPUUtilization-25.0), 1e-5)
				})
			})
		})

	})
}
