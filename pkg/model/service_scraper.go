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
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/serving/pkg/autoscaler"
	"math/rand"
	"time"
)

type clusterServiceScraper struct {
	replicasActive ReplicasActiveStock
}

func (cm *clusterServiceScraper) Scrape() (*autoscaler.StatMessage, error) {
	replicaCount := cm.replicasActive.Count()

	if replicaCount > 0 {
		var (
			now                   *time.Time
			avgConcurrency        float64
			avgProxiedConcurrency float64
			reqCount              float64
			proxiedReqCount       float64
		)

		idx := 0
		if replicaCount > 3 {
			idx = 3
		} else {
			idx = int(replicaCount)
		}

		for i := 0; i < idx; i++ {
			replicas := cm.replicasActive.EntitiesInStock()
			rep := (*replicas[rand.Intn(len(replicas))]).(ReplicaEntity)
			stat := rep.Stat()

			now = stat.Time
			avgConcurrency += stat.AverageConcurrentRequests
			avgProxiedConcurrency += stat.AverageProxiedConcurrentRequests
			reqCount += stat.RequestCount
			proxiedReqCount += stat.ProxiedRequestCount
		}

		return &autoscaler.StatMessage{
			Key: types.NamespacedName{Namespace: testName, Name: testName},
			Stat: autoscaler.Stat{
				Time:                             now,
				PodName:                          "averaged-pods",
				AverageConcurrentRequests:        avgConcurrency,
				AverageProxiedConcurrentRequests: avgProxiedConcurrency,
				RequestCount:                     reqCount,
				ProxiedRequestCount:              proxiedReqCount,
			},
		}, nil
	}

	return nil, nil
}

func NewClusterServiceScraper(stock ReplicasActiveStock) autoscaler.StatsScraper {
	return &clusterServiceScraper{
		replicasActive: stock,
	}
}
