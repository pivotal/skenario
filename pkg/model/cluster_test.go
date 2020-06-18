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
	"skenario/pkg/simulator"
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers/core/v1"
)

func TestCluster(t *testing.T) {
	spec.Run(t, "Cluster model", testCluster, spec.Report(report.Terminal{}))
	spec.Run(t, "EPInformer interface", testEPInformer, spec.Report(report.Terminal{}))
}

func testCluster(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var subject ClusterModel
	var rawSubject *clusterModel
	var envFake *FakeEnvironment
	var endpoints *corev1.Endpoints
	var err error
	var replicasConfig ReplicasConfig

	it.Before(func() {
		config = ClusterConfig{}
		config.NumberOfRequests = 10
		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		subject = NewCluster(envFake, config, replicasConfig)
		assert.NotNil(t, subject)

		rawSubject = subject.(*clusterModel)
		endpoints, err = rawSubject.kubernetesClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
		assert.NoError(t, err)
	})

	describe("NewCluster()", func() {
		envFake = new(FakeEnvironment)

		it("sets an environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})

		it("creates an 'empty' Endpoints entry for 'Skenario Revision'", func() {
			assert.Equal(t, "Skenario Revision", endpoints.Name)
			assert.Len(t, endpoints.Subsets, 1)
			assert.Len(t, endpoints.Subsets[0].Addresses, 0)
		})
	})

	describe("Desired()", func() {
		it("returns the ReplicasDesired stock", func() {
			assert.Equal(t, rawSubject.replicasDesired, subject.Desired())
		})
	})

	describe("CurrentLaunching()", func() {
		envFake = new(FakeEnvironment)

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "11.11.11.11", &failedSink)
			secondReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "22.22.22.22", &failedSink)
			rawSubject.replicasLaunching.Add(firstReplica)
			rawSubject.replicasLaunching.Add(secondReplica)
		})

		it("gives the .Count() of replicas launching", func() {
			assert.Equal(t, uint64(2), subject.CurrentLaunching())
		})
	})

	describe("CurrentActive()", func() {
		envFake = new(FakeEnvironment)

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "11.11.11.11", &failedSink)
			secondReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "22.22.22.22", &failedSink)
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
		var routingStockRecorded autoscaler.Stat
		var theTime = time.Now()
		var replicaFake *FakeReplica
		envFake = new(FakeEnvironment)
		recordOnce := 1
		recordThrice := 3

		it.Before(func() {
			rawSubject = subject.(*clusterModel)

			replicaFake = new(FakeReplica)

			autoscalerFake = &fakeAutoscaler{
				recorded:   make([]autoscaler.Stat, 0),
				scaleTimes: make([]time.Time, 0),
			}

			request := NewRequestEntity(envFake, rawSubject.requestsInRouting, RequestConfig{CPUTimeMillis: 500, IOTimeMillis: 500, Timeout: 1 * time.Second})
			rawSubject.requestsInRouting.Add(request)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "11.11.11.11", &failedSink)
			secondReplica := NewReplicaEntity(envFake, rawSubject.kubernetesClient, rawSubject.endpointsInformer, "22.22.22.22", &failedSink)

			rawSubject.replicasActive.Add(replicaFake)

			rawSubject.replicasActive.Add(firstReplica)
			rawSubject.replicasActive.Add(secondReplica)

			subject.RecordToAutoscaler(autoscalerFake, &theTime)
			routingStockRecorded = autoscalerFake.recorded[0]
		})

		// TODO immediately record arrivals at routingStock

		it("records once for the routingStock and once each replica in ReplicasActive", func() {
			assert.Len(t, autoscalerFake.recorded, recordOnce+recordThrice)
		})

		describe("the record for the routingStock", func() {
			it("sets time to the movement OccursAt", func() {
				assert.Equal(t, &theTime, routingStockRecorded.Time)
			})

			it("sets the PodName to 'RoutingStock'", func() {
				assert.Equal(t, "RoutingStock", routingStockRecorded.PodName)
			})

			it("sets AverageConcurrentRequests to the number of Requests in the routingStock", func() {
				assert.Equal(t, 1.0, routingStockRecorded.AverageConcurrentRequests)
			})

			it("sets RequestCount to the net change in the number of Requests since last invocation", func() {
				assert.Equal(t, int32(1), routingStockRecorded.RequestCount)
			})
		})

		describe("records for replicas", func() {
			it("delegates Stat creation to the Replica", func() {
				assert.True(t, replicaFake.StatCalled)
			})
		})
	})

	describe("requestsInRouting", func() {
		it("returns the configured routing stock", func() {
			assert.Equal(t, rawSubject.requestsInRouting, subject.RoutingStock())
		})
	})
}

func testEPInformer(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var subject EndpointInformerSource
	var cluster ClusterModel
	var envFake = new(FakeEnvironment)
	var replicasConfig ReplicasConfig

	it.Before(func() {
		config = ClusterConfig{}
		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		cluster = NewCluster(envFake, config, replicasConfig)
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
