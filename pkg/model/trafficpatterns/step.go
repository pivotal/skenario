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

type step struct {
	env       simulator.Environment
	rps       int
	stepAfter time.Duration
	source    model.TrafficSource
	buffer    model.RequestsBufferedStock
}

type StepConfig struct {
	RPS       int           `json:"rps"`
	StepAfter time.Duration `json:"step_after"`
}

func (*step) Name() string {
	return "step"
}

func (s *step) Generate() {
	var t time.Time
	startAt := s.env.CurrentMovementTime().Add(s.stepAfter)

	for t = startAt; t.Before(s.env.HaltTime()); t = t.Add(1 * time.Second) {
		uniRand := NewUniformRandom(s.env, s.source, s.buffer, UniformConfig{
			NumberOfRequests: s.rps,
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
	}
}

func NewStep(env simulator.Environment, source model.TrafficSource, buffer model.RequestsBufferedStock, config StepConfig) Pattern {
	return &step{
		env:       env,
		rps:       config.RPS,
		stepAfter: config.StepAfter,
		source:    source,
		buffer:    buffer,
	}
}
