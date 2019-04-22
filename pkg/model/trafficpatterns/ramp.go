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
	var t time.Time
	nextAdd := r.increaseRate
	startAt := r.env.CurrentMovementTime()
	rampUpDuration := r.env.HaltTime().Sub(r.env.CurrentMovementTime()) / 2
	downAt := startAt.Add(rampUpDuration.Round(time.Second))

	for t = startAt; t.Before(downAt); t = t.Add(1 * time.Second) {
		uniRand := NewUniformRandom(r.env, r.source, r.buffer, nextAdd, t, 1*time.Second)
		uniRand.Generate()
		nextAdd = nextAdd + r.increaseRate
	}

	for ; t.Before(r.env.HaltTime()); t = t.Add(1 * time.Second) {
		nextAdd = nextAdd - r.increaseRate
		uniRand := NewUniformRandom(r.env, r.source, r.buffer, nextAdd, t, 1*time.Second)
		uniRand.Generate()
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
