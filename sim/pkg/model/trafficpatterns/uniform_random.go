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

package trafficpatterns

import (
	"math/rand"
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type uniformRandom struct {
	env              simulator.Environment
	source           model.TrafficSource
	routingStock     model.RequestsRoutingStock
	numberOfRequests int
	startAt          time.Time
	runFor           time.Duration
}

type UniformConfig struct {
	NumberOfRequests int           `json:"number_of_requests"`
	StartAt          time.Time     `json:"start_at"`
	RunFor           time.Duration `json:"run_for"`
}

func (ur *uniformRandom) Name() string {
	return "golang_rand_uniform"
}

func (ur *uniformRandom) Generate() {
	for i := 0; i < ur.numberOfRequests; i++ {
		r := rand.Int63n(ur.runFor.Nanoseconds())

		ur.env.AddToSchedule(simulator.NewMovement(
			"arrive_at_routing_stock",
			ur.startAt.Add(time.Duration(r)*time.Nanosecond),
			ur.source,
			ur.routingStock,
			nil,
		))
	}
}

func NewUniformRandom(env simulator.Environment, source model.TrafficSource, routingStock model.RequestsRoutingStock, config UniformConfig) Pattern {
	return &uniformRandom{
		env:              env,
		source:           source,
		routingStock:     routingStock,
		numberOfRequests: config.NumberOfRequests,
		startAt:          config.StartAt,
		runFor:           config.RunFor,
	}
}
