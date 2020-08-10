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

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "HPA Autoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

func testAutoscaler(t *testing.T, describe spec.G, it spec.S) {
	var subject AutoscalerModel
	var rawSubject *autoscaler
	var envFake *FakeEnvironment
	var cluster ClusterModel
	var config ClusterConfig
	var replicasConfig ReplicasConfig
	startAt := time.Unix(0, 0)

	it.Before(func() {
		config = ClusterConfig{}
		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		envFake = &FakeEnvironment{
			Movements:          make([]simulator.Movement, 0),
			TheTime:            startAt,
			TheHaltTime:        startAt.Add(1 * time.Hour),
			ThePluginPartition: NewFakePluginPartition(),
		}
		cluster = NewCluster(envFake, config, replicasConfig)
	})

	describe("NewAutoscaler()", func() {
		it.Before(func() {
			subject = NewAutoscaler(envFake, startAt, cluster, AutoscalerConfig{TickInterval: 60 * time.Second})
			rawSubject = subject.(*autoscaler)
		})

		describe("scheduling calculations and waits", func() {
			var tickInterval time.Duration
			var tickMovements []simulator.Movement

			it.Before(func() {
				tickInterval = 1 * time.Minute
				tickMovements = []simulator.Movement{}

				for _, mv := range envFake.Movements {
					if mv.Kind() == "autoscaler_tick" {
						tickMovements = append(tickMovements, mv)
					}
				}
			})

			it("schedules an autoscaler_tick movement to occur on each TickInterval", func() {
				assert.Len(t, tickMovements, 59)

				theTime := startAt.Add(1 * time.Nanosecond)
				for _, mv := range tickMovements {
					theTime = theTime.Add(tickInterval)

					assert.Equal(t, theTime, mv.OccursAt())
				}
			})
		})

		it("sets an Environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})

		it("sets a ticktock stock", func() {
			assert.NotNil(t, rawSubject.tickTock)
			assert.Equal(t, simulator.StockName("Autoscaler Ticktock"), rawSubject.tickTock.Name())
		})
	})
}
