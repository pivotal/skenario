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
	"knative.dev/serving/pkg/autoscaler"
)

func TestClusterServiceScraper(t *testing.T) {
	spec.Run(t, "Cluster Service Scraper", testServiceScraper, spec.Report(report.Terminal{}))
}

func testServiceScraper(t *testing.T, describe spec.G, it spec.S) {
	var (
		subject        autoscaler.StatsScraper
		rawSubject     *clusterServiceScraper
		replicasActive ReplicasActiveStock
	)

	it.Before(func() {
		replicasActive = NewReplicasActiveStock()
		subject = NewClusterServiceScraper(replicasActive)
		rawSubject = subject.(*clusterServiceScraper)
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
	})
}
