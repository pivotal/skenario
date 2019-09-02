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

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"knative.dev/serving/pkg/autoscaler"

	"skenario/pkg/simulator"
)

func TestClusterServiceScraper(t *testing.T) {
	spec.Run(t, "Cluster Service Scraper", testServiceScraper, spec.Report(report.Terminal{}))
}

func testServiceScraper(t *testing.T, describe spec.G, it spec.S) {
	var (
		subject        autoscaler.StatsScraper
		rawSubject     *clusterServiceScraper
		replicasActive ReplicasActiveStock
		envFake        *FakeEnvironment
	)

	it.Before(func() {
		replicasActive = NewReplicasActiveStock()
		subject = NewClusterServiceScraper(replicasActive)
		rawSubject = subject.(*clusterServiceScraper)
		envFake = new(FakeEnvironment)
	})

	describe("NewClusterServiceScraper()", func() {
		it("sets replicas active", func() {
			assert.NotNil(t, rawSubject.replicasActive)
			assert.Equal(t, rawSubject.replicasActive, replicasActive)
		})
	})

	describe("Scrape()", func() {
		describe("when there are now replicas active", func() {
			var (
				statMsg *autoscaler.StatMessage
				err     error
			)

			it.Before(func() {
				statMsg, err = subject.Scrape()
			})

			it("returns nothing", func() {
				assert.Nil(t, statMsg)
				assert.NoError(t, err)
			})
		})

		describe("when there are <= 3 replicas active", func() {
			var (
				rawReplica *replicaEntity
				statMsg    *autoscaler.StatMessage
				err error
			)

			it.Before(func() {
				replica := NewReplicaEntity(envFake, 10)
				rawReplica = replica.(*replicaEntity)

				rawReplica.requestsProcessing.Add(simulator.NewEntity("Test Processing Request", "Request"))
				rawReplica.requestsComplete.Add(simulator.NewEntity("Test Complete Request", "Request"))
				replicasActive.Add(replica)

				statMsg, err = subject.Scrape()
				require.NoError(t, err)
			})

			//describe("the Key", func() {
			//	var key types.NamespacedName
			//
			//	it.Before(func() {
			//		key = statMsg.Key
			//	})
			//
			//	it("uses the common values key", func() {
			//		assert.Equal(t, statKey, key)
			//	})
			//})

			describe("the Stat", func() {
				var stat autoscaler.Stat

				it.Before(func() {
					stat = statMsg.Stat
				})

				it("takes stats from all of the replicas", func() {
					assert.Equal(t, 1, stat.AverageConcurrentRequests)
					assert.Equal(t, 1, stat.AverageProxiedConcurrentRequests)
					assert.Equal(t, 1, stat.RequestCount)
					assert.Equal(t, 1, stat.ProxiedRequestCount)
				})
			})

		})

		// replica count <= 3
		// sample all of the entities

		// replica count > 3
		// sample at random

		// rep.Stat()

		// on success returns aggregated values
		// key is namespaced name ... which gets there how??

		// "returns nil, nil"

	})
}
