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
 *
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
	buffer           model.RequestsBufferedStock
	numberOfRequests int
}

func (ur *uniformRandom) Name() string {
	return "golang_rand_uniform"
}

func (ur *uniformRandom) Generate() {
	runsFor := ur.env.HaltTime().Sub(ur.env.CurrentMovementTime())
	for i := 0; i < ur.numberOfRequests; i++ {
		r := rand.Int63n(runsFor.Nanoseconds())

		ur.env.AddToSchedule(simulator.NewMovement(
			"arrive_at_buffer",
			ur.env.CurrentMovementTime().Add(time.Duration(r)*time.Nanosecond),
			ur.source,
			ur.buffer,
		))
	}
}

func NewUniformRandom(env simulator.Environment, source model.TrafficSource, buffer model.RequestsBufferedStock, numberOfRequests int) Pattern {
	return &uniformRandom{
		env:    env,
		source: source,
		buffer: buffer,
		numberOfRequests: numberOfRequests,
	}
}
