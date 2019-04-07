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
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/simulator"
)

func TestAutoscalerTicktock(t *testing.T) {
	spec.Run(t, "Ticktock stock", testAutoscalerTicktock, spec.Report(report.Terminal{}))
}

func testAutoscalerTicktock(t *testing.T, describe spec.G, it spec.S) {
	var subject AutoscalerTicktockStock
	var rawSubject *autoscalerTicktockStock
	var envFake *fakeEnvironment
	var autoscalerFake *fakeAutoscaler
	var cluster ClusterModel

	it.Before(func() {
		envFake = new(fakeEnvironment)
		envFake.theTime = time.Unix(0, 0)
		autoscalerFake = &fakeAutoscaler{
			recorded:   make([]autoscaler.Stat, 0),
			scaleTimes: make([]time.Time, 0),
		}
		cluster = NewCluster(envFake, ClusterConfig{})
		subject = NewAutoscalerTicktockStock(envFake, simulator.NewEntity("Autoscaler", "KnativeAutoscaler"), autoscalerFake, cluster)
		rawSubject = subject.(*autoscalerTicktockStock)
	})

	describe("NewAutoscalerTicktockStock()", func() {
		it("sets the entity", func() {
			assert.Equal(t, simulator.EntityName("Autoscaler"), rawSubject.autoscalerEntity.Name())
			assert.Equal(t, simulator.EntityKind("KnativeAutoscaler"), rawSubject.autoscalerEntity.Kind())
		})
	})

	describe("Name()", func() {
		it("is called 'KnativeAutoscaler Stock'", func() {
			assert.Equal(t, subject.Name(), simulator.StockName("Autoscaler Ticktock"))
		})
	})

	describe("KindStocked()", func() {
		it("accepts Knative Autoscalers", func() {
			assert.Equal(t, subject.KindStocked(), simulator.EntityKind("KnativeAutoscaler"))
		})
	})

	describe("Count()", func() {
		it("always has 1 entity stocked", func() {
			assert.Equal(t, subject.Count(), uint64(1))

			ent := subject.Remove()
			err := subject.Add(ent)
			assert.NoError(t, err)
			err = subject.Add(ent)
			assert.NoError(t, err)

			assert.Equal(t, subject.Count(), uint64(1))

			subject.Remove()
			subject.Remove()
			subject.Remove()
			assert.Equal(t, subject.Count(), uint64(1))
		})
	})

	describe("Remove()", func() {
		it("gives back the one KnativeAutoscaler", func() {
			assert.Equal(t, subject.Remove(), subject.Remove())
		})
	})

	describe("Add()", func() {
		describe("ensuring consistency", func() {
			var differentEntity simulator.Entity

			it.Before(func() {
				differentEntity = simulator.NewEntity("Different!", "KnativeAutoscaler")
			})

			it("returns error if the Added entity does not equal the existing entity", func() {
				err := subject.Add(differentEntity)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "different from the entity given at creation time")
			})
		})

		describe("driving the Knative autoscaler", func() {
			describe("controlling time", func() {
				it.Before(func() {
					ent := subject.Remove()
					err := subject.Add(ent)
					assert.NoError(t, err)
				})

				it("triggers the autoscaler calculation with the current time", func() {
					assert.Equal(t, time.Unix(0, 0), autoscalerFake.scaleTimes[0])
				})
			})

			describe("updating statistics", func() {
				var rawCluster *clusterModel
				onceForBuffer := 1
				onceForReplica := 1

				it.Before(func() {
					rawCluster = cluster.(*clusterModel)
					newReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "22.22.22.22")
					err := rawCluster.replicasActive.Add(newReplica)
					assert.NoError(t, err)

					ent := subject.Remove()
					err = subject.Add(ent)
					assert.NoError(t, err)
				})

				it("delegates statistics updating to ClusterModel", func() {
					assert.Len(t, autoscalerFake.recorded, onceForBuffer+onceForReplica)
				})
			})

			describe("the autoscaler was able to make a recommendation", func() {
				var rawCluster *clusterModel
				var activeBefore uint64

				it.Before(func() {
					autoscalerFake.scaleTo = 2

					rawCluster = cluster.(*clusterModel)
					newReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "33.33.33.33")
					err := rawCluster.replicasActive.Add(newReplica)
					assert.NoError(t, err)

					activeBefore = rawSubject.cluster.CurrentActive()
					ent := subject.Remove()
					err = subject.Add(ent)
					assert.NoError(t, err)
				})

				it("sets the current desired on the cluster", func() {
					assert.NotEqual(t, activeBefore, rawSubject.cluster.CurrentDesired())
					assert.Equal(t, int32(2), rawSubject.cluster.CurrentDesired())
				})
			})

			describe.Pend("the autoscaler failed to make a recommendation", func() {
				it.Before(func() {
					autoscalerFake.cantDecide = true

				})

				it("notes that there was a problem", func() {

				})
			})
		})

	})
}
