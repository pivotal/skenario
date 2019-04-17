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
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type ramp struct {
	env          simulator.Environment
	source       model.TrafficSource
	buffer       model.RequestsBufferedStock
	increaseRate int
}

func (*ramp) Name() string {
	return "ramp"
}

func (r *ramp) Generate() {
	nextAdd := r.increaseRate

	startAt := r.env.CurrentMovementTime().Add(1*time.Second)
	for t := startAt; t.Before(r.env.HaltTime()); t = t.Add(1 * time.Second) {
		for i := 1; i <= nextAdd; i++ {
			r.env.AddToSchedule(simulator.NewMovement(
				"arrive_at_buffer",
				t.Add(time.Duration(i)*time.Nanosecond),
				r.source,
				r.buffer,
			))
		}
		nextAdd = nextAdd + r.increaseRate
	}
}

func NewRamp(env simulator.Environment, source model.TrafficSource, buffer model.RequestsBufferedStock, increaseRate int) Pattern {
	return &ramp{
		env:          env,
		source:       source,
		buffer:       buffer,
		increaseRate: increaseRate,
	}
}
