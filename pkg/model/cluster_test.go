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
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers/core/v1"

	"knative-simulator/pkg/simulator"
)

func TestCluster(t *testing.T) {
	spec.Run(t, "Cluster model", testCluster, spec.Report(report.Terminal{}))
	spec.Run(t, "EPInformer interface", testEPInformer, spec.Report(report.Terminal{}))
	spec.Run(t, "IPV4Sequence interface", testIPV4Sequence, spec.Report(report.Terminal{}))
}

func testCluster(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var subject ClusterModel
	var rawSubject *clusterModel
	var envFake = new(fakeEnvironment)
	var endpoints *corev1.Endpoints
	var err error

	it.Before(func() {
		config = ClusterConfig{}
		subject = NewCluster(envFake, config)
		assert.NotNil(t, subject)

		rawSubject = subject.(*clusterModel)
		endpoints, err = rawSubject.kubernetesClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
		assert.NoError(t, err)
	})

	describe("NewCluster()", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})

		it("creates an 'empty' Endpoints entry for 'Skenario Revision'", func() {
			assert.Equal(t, "Skenario Revision", endpoints.Name)
			assert.Len(t, endpoints.Subsets, 1)
			assert.Len(t, endpoints.Subsets[0].Addresses, 0)
		})
	})

	describe("CurrentDesired()", func() {
		it("defaults to 0", func() {
			assert.Equal(t, int32(0), subject.CurrentDesired())
		})
	})

	describe("SetDesired()", func() {
		describe("using ClusterConfig delay values", func() {
			var firstLaunchAt, secondLaunchAt time.Time
			var firstTerminateAt, secondTerminateAt, thirdTerminateAt, fourthTerminateAt time.Time
			config.LaunchDelay = 11*time.Second
			config.TerminateDelay = 22*time.Second

			describe.Focus("ClusterConfig.LaunchDelay", func() {
				it.Before(func() {
					envFake.theTime = time.Unix(0, 0)

					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					firstLaunchAt = envFake.theTime.Add(rawSubject.config.LaunchDelay)
					secondLaunchAt = firstLaunchAt.Add(1*time.Nanosecond)

					subject.SetDesired(2)
				})

				it("delays the first replica by the exact value", func() {
					assert.Equal(t, firstLaunchAt, envFake.movements[0].OccursAt())
				})

				it("adds a nanosecond to each subsequent replica launch to prevent collisions", func() {
					assert.Equal(t, secondLaunchAt, envFake.movements[1].OccursAt())
				})
			})

			describe("ClusterConfig.TerminateDelay", func() {
				it.Before(func() {
					envFake.theTime = time.Unix(0, 0)

					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching #1", simulator.EntityKind("Replica")))
					assert.NoError(t, err)
					err = rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching #2", simulator.EntityKind("Replica")))
					assert.NoError(t, err)
					err = rawSubject.replicasActive.Add(NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next()))
					assert.NoError(t, err)
					err = rawSubject.replicasActive.Add(NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next()))
					assert.NoError(t, err)

					firstTerminateAt = envFake.theTime.Add(rawSubject.config.TerminateDelay)
					secondTerminateAt = firstTerminateAt.Add(1 * time.Nanosecond)
					thirdTerminateAt = secondTerminateAt.Add(1 * time.Nanosecond)
					fourthTerminateAt = thirdTerminateAt.Add(1 * time.Nanosecond)

					subject.SetDesired(0)
				})

				it("delays the termination of the first launching replica by the exact amount", func() {
					assert.Equal(t, firstTerminateAt, envFake.movements[0].OccursAt())
				})

				it("delays the termination of the second launching replica by an additional nanosecond", func() {
					assert.Equal(t, secondTerminateAt, envFake.movements[1].OccursAt())
				})

				it("delays the termination of each active replicas by a nanosecond", func() {
					// two examples to force a full pass through the loop for terminating active replicas
					assert.Equal(t, thirdTerminateAt, envFake.movements[2].OccursAt())
					assert.Equal(t, fourthTerminateAt, envFake.movements[3].OccursAt())
				})
			})
		})

		describe("there are launching replicas but no active replicas", func() {
			describe("new value > launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching", simulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(9)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(9), subject.CurrentDesired())
				})

				it("Adds replica entities to ReplicasLaunching to bring them up to desired", func() {
					assert.Equal(t, uint64(9), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 8)
					assert.Equal(t, simulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching", simulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasLaunching to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, simulator.MovementKind("launching -> terminated"), envFake.movements[0].Kind())
				})
			})

			describe("new value == launching replicas", func() {
				it.Before(func() {
					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					err := rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching 1", simulator.EntityKind("Replica")))
					assert.NoError(t, err)
					err = rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching 2", simulator.EntityKind("Replica")))
					assert.NoError(t, err)

					subject.SetDesired(2)
					subject.SetDesired(2)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
					assert.Equal(t, uint64(2), rawSubject.replicasLaunching.Count())
				})
			})
		})

		describe("there are active replicas but no launching replicas", func() {
			it.Before(func() {
				subject = NewCluster(envFake, config)
				assert.NotNil(t, subject)

				rawSubject = subject.(*clusterModel)
				envFake.movements = make([]simulator.Movement, 0)

				newReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
				err := rawSubject.replicasActive.Add(newReplica)
				assert.NoError(t, err)
			})

			describe("new value > active replicas", func() {
				it.Before(func() {
					subject.SetDesired(2)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
				})

				it("Adds one replica entity to ReplicasLaunching to close the gap between ReplicasActive and desired", func() {
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, simulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < active replicas", func() {
				it.Before(func() {
					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, simulator.MovementKind("active -> terminated"), envFake.movements[0].Kind())
				})
			})

			describe("new value == active replicas", func() {
				it.Before(func() {
					subject.SetDesired(1)
					subject.SetDesired(1)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(1), subject.CurrentDesired())
					assert.Equal(t, uint64(1), rawSubject.replicasActive.Count())
				})
			})
		})

		describe("there is a mix of active and launching replicas", func() {
			it.Before(func() {
				subject = NewCluster(envFake, config)
				assert.NotNil(t, subject)

				rawSubject = subject.(*clusterModel)
				envFake.movements = make([]simulator.Movement, 0)

				newReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
				err := rawSubject.replicasActive.Add(newReplica)
				assert.NoError(t, err)
				err = rawSubject.replicasLaunching.Add(simulator.NewEntity("already launching", simulator.EntityKind("Replica")))
				assert.NoError(t, err)
			})

			describe("new value > active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(3)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(3), subject.CurrentDesired())
				})

				it("Adds another replica entity to ReplicasLaunching to close the gap between ReplicasLaunching + ReplicasActive and desired", func() {
					assert.Equal(t, uint64(2), rawSubject.replicasLaunching.Count())
				})

				it("adds another movement from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, simulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})
			})

			describe("new value < active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(0)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(0), subject.CurrentDesired())
				})

				it("schedules movements from ReplicasActive to ReplicasTerminating", func() {
					assert.Len(t, envFake.movements, 2)
					assert.Equal(t, "launching -> terminated", string(envFake.movements[0].Kind()))
					assert.Equal(t, "active -> terminated", string(envFake.movements[1].Kind()))
				})
			})

			describe("new value == active replicas + launching replicas", func() {
				it.Before(func() {
					subject.SetDesired(2)
					subject.SetDesired(2)
				})

				it("doesn't change anything", func() {
					assert.Equal(t, int32(2), subject.CurrentDesired())
					assert.Equal(t, uint64(1), rawSubject.replicasActive.Count())
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})
			})
		})

		describe("there are no active or launching replicas", func() {
			describe("new value > 0", func() {
				it.Before(func() {
					subject = NewCluster(envFake, config)
					assert.NotNil(t, subject)

					rawSubject = subject.(*clusterModel)
					envFake.movements = make([]simulator.Movement, 0)

					subject.SetDesired(1)
				})

				it("updates the number of desired replicas", func() {
					assert.Equal(t, int32(1), subject.CurrentDesired())
				})

				it("Adds replica entities to ReplicasLaunching to bring them up to desired", func() {
					assert.Equal(t, uint64(1), rawSubject.replicasLaunching.Count())
				})

				it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
					assert.Len(t, envFake.movements, 1)
					assert.Equal(t, simulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
				})

			})
		})
	})

	describe("CurrentLaunching()", func() {
		it.Before(func() {
			subject.SetDesired(7)
		})

		it("gives the .Count() of replicas launching", func() {
			assert.Equal(t, uint64(7), subject.CurrentLaunching())
		})
	})

	describe("CurrentActive()", func() {
		var rawSubject *clusterModel

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			firstReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
			secondReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
			rawSubject.replicasActive.Add(firstReplica)
			rawSubject.replicasActive.Add(secondReplica)
		})

		it("gives the .Count() of replicas active", func() {
			assert.Equal(t, uint64(2), subject.CurrentActive())
		})
	})

	describe("RecordToAutoscaler()", func() {
		var autoscalerFake *fakeAutoscaler
		var rawSubject *clusterModel
		var firstRecorded autoscaler.Stat
		var theTime = time.Now()
		var ctx = context.Background()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)

			autoscalerFake = &fakeAutoscaler{
				recorded:   make([]autoscaler.Stat, 0),
				scaleTimes: make([]time.Time, 0),
			}

			firstReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
			secondReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())
			thirdReplica := NewReplicaEntity(rawSubject.kubernetesClient, rawSubject.endpointsInformer, rawSubject.Next())

			rawSubject.replicasActive.Add(firstReplica)
			rawSubject.replicasActive.Add(secondReplica)
			rawSubject.replicasActive.Add(thirdReplica)

			subject.RecordToAutoscaler(autoscalerFake, &theTime, ctx)
			firstRecorded = autoscalerFake.recorded[0]
		})

		describe("Records added to the Autoscaler", func() {
			it("records once for each replica in ReplicasActive", func() {
				assert.Len(t, autoscalerFake.recorded, 3)
			})

			it("sets time to the movement OccursAt", func() {
				assert.Equal(t, &theTime, firstRecorded.Time)
			})

			it("sets the PodName to Replica name", func() {
				assert.Equal(t, "Replica", firstRecorded.PodName)
			})

			it("sets AverageConcurrentRequests to 1", func() {
				assert.Equal(t, float64(1.0), firstRecorded.AverageConcurrentRequests)
			})

			it("sets RequestCount to 1", func() {
				assert.Equal(t, int32(1), firstRecorded.RequestCount)
			})
		})
	})
}

func testEPInformer(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var subject EndpointInformerSource
	var cluster ClusterModel
	var envFake = new(fakeEnvironment)

	it.Before(func() {
		config = ClusterConfig{}
		cluster = NewCluster(envFake, config)
		assert.NotNil(t, cluster)
		subject = cluster.(EndpointInformerSource)
		assert.NotNil(t, subject)
	})

	describe("EPInformer()", func() {
		// TODO: this test just feels like it's testing the compiler
		it("returns an EndpointsInformer", func() {
			assert.Implements(t, (*v1.EndpointsInformer)(nil), subject.EPInformer())
		})
	})
}

func testIPV4Sequence(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var cluster ClusterModel
	var subject IPV4Sequence
	var rawSubject *clusterModel
	var envFake = new(fakeEnvironment)

	it.Before(func() {
		config = ClusterConfig{}
		cluster = NewCluster(envFake, config)
		subject = cluster.(IPV4Sequence)
		rawSubject = cluster.(*clusterModel)
	})

	describe("NextIP()", func() {
		var ipGiven string
		it.Before(func() {
			// twice to show we didn't succeed purely on init values
			ipGiven = subject.Next()
			ipGiven = subject.Next()
		})

		it("creates an IPv4 address string", func() {
			assert.Equal(t, "0.0.0.2", ipGiven)
		})

		it("increments the next IP to give out", func() {
			assert.Equal(t, uint32(3), rawSubject.nextIPValue)
		})
	})

}
