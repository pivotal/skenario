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
	"skenario/pkg/simulator"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestCluster(t *testing.T) {
	spec.Run(t, "Cluster model", testCluster, spec.Report(report.Terminal{}))
}

func testCluster(t *testing.T, describe spec.G, it spec.S) {
	var config ClusterConfig
	var subject ClusterModel
	var rawSubject *clusterModel
	var envFake *FakeEnvironment
	var err error
	var replicasConfig ReplicasConfig

	it.Before(func() {
		config = ClusterConfig{}
		config.NumberOfRequests = 10
		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		subject = NewCluster(envFake, config, replicasConfig)
		assert.NotNil(t, subject)

		rawSubject = subject.(*clusterModel)
		assert.NoError(t, err)
	})

	describe("NewCluster()", func() {
		envFake = NewFakeEnvironment()

		it("sets an environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})
	})

	describe("Desired()", func() {
		it("returns the ReplicasDesired stock", func() {
			assert.Equal(t, rawSubject.replicasDesired, subject.Desired())
		})
	})

	describe("CurrentLaunching()", func() {
		envFake = NewFakeEnvironment()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, &failedSink)
			secondReplica := NewReplicaEntity(envFake, &failedSink)
			rawSubject.replicasLaunching.Add(firstReplica)
			rawSubject.replicasLaunching.Add(secondReplica)
		})

		it("gives the .Count() of replicas launching", func() {
			assert.Equal(t, uint64(2), subject.CurrentLaunching())
		})
	})

	describe("CurrentActive()", func() {
		envFake = NewFakeEnvironment()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, &failedSink)
			secondReplica := NewReplicaEntity(envFake, &failedSink)
			rawSubject.replicasActive.Add(firstReplica)
			rawSubject.replicasActive.Add(secondReplica)
		})

		it("gives the .Count() of replicas active", func() {
			assert.Equal(t, uint64(2), subject.CurrentActive())
		})
	})

	describe("RecordToAutoscaler()", func() {
		var rawSubject *clusterModel
		var routingStockRecorded proto.Stat
		var theTime = time.Now()
		envFake = NewFakeEnvironment()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)

			request := NewRequestEntity(envFake, rawSubject.requestsInRouting, RequestConfig{CPUTimeMillis: 500, IOTimeMillis: 500, Timeout: 1 * time.Second})
			rawSubject.requestsInRouting.Add(request)
			failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
			firstReplica := NewReplicaEntity(envFake, &failedSink)
			secondReplica := NewReplicaEntity(envFake, &failedSink)

			rawSubject.replicasActive.Add(firstReplica)
			rawSubject.replicasActive.Add(secondReplica)

			subject.RecordToAutoscaler(&theTime)
			routingStockRecorded = *envFake.ThePlugin.(*FakePluginPartition).stats[0]
		})

		// TODO immediately record arrivals at routingStock

		it("records once for the routingStock and twice for each replica in ReplicasActive, we have 2 replicas", func() {
			stats := envFake.ThePlugin.(*FakePluginPartition).stats
			assert.Len(t, envFake.ThePlugin.(*FakePluginPartition).stats, 5)
			assert.Equal(t, stats[0].Type, proto.MetricType_CONCURRENT_REQUESTS_MILLIS)
			assert.Equal(t, stats[1].Type, proto.MetricType_CONCURRENT_REQUESTS_MILLIS)
			assert.Equal(t, stats[2].Type, proto.MetricType_CPU_MILLIS)
			assert.Equal(t, stats[3].Type, proto.MetricType_CONCURRENT_REQUESTS_MILLIS)
			assert.Equal(t, stats[4].Type, proto.MetricType_CPU_MILLIS)

		})

		describe("the record for the routingStock", func() {
			it("sets time to the movement OccursAt", func() {
				assert.Equal(t, theTime.UnixNano(), routingStockRecorded.Time)
			})

			it("sets the PodName to 'RoutingStock'", func() {
				assert.Equal(t, "RoutingStock", routingStockRecorded.PodName)
			})

			it("sets Value to the number of Requests in the routingStock*1000", func() {
				assert.Equal(t, int32(1000), routingStockRecorded.Value)
			})
		})
	})

	describe("requestsInRouting", func() {
		it("returns the configured routing stock", func() {
			assert.Equal(t, rawSubject.requestsInRouting, subject.RoutingStock())
		})
	})
}
