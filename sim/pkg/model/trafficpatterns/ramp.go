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
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type ramp struct {
	env          simulator.Environment
	source       model.TrafficSource
	routingStock model.RequestsRoutingStock
	sink         model.RequestsProcessingStock
	deltaV       float64
	maxRPS       int
}

type RampConfig struct {
	DeltaV float64 `json:"delta_v"`
	MaxRPS int     `json:"max_rps"`
}

func (*ramp) Name() string {
	return "ramp"
}

func (r *ramp) Generate() {
	var t time.Time
	nextRPS := r.deltaV
	startAt := r.env.CurrentMovementTime()

	for t = startAt; int(nextRPS) <= r.maxRPS; t = t.Add(1 * time.Second) {
		uniRand := NewUniformRandom(r.env, r.source, r.routingStock, UniformConfig{
			NumberOfRequests: int(nextRPS),
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
		nextRPS = nextRPS + r.deltaV
	}

	for ; nextRPS > 0; t = t.Add(1 * time.Second) {
		nextRPS = nextRPS - r.deltaV
		uniRand := NewUniformRandom(r.env, r.source, r.routingStock, UniformConfig{
			NumberOfRequests: int(nextRPS),
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
	}
}

func NewRamp(env simulator.Environment, source model.TrafficSource, routingStock model.RequestsRoutingStock, config RampConfig) Pattern {
	return &ramp{
		env:          env,
		source:       source,
		routingStock: routingStock,
		deltaV:       config.DeltaV,
		maxRPS:       config.MaxRPS,
	}
}
