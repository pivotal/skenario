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
	"math"
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type sinusoidal struct {
	env       simulator.Environment
	amplitude int
	period    time.Duration
	source    model.TrafficSource
	buffer    model.RequestsRoutingStock
}

type SinusoidalConfig struct {
	Amplitude int           `json:"amplitude"`
	Period    time.Duration `json:"period"`
}

func (*sinusoidal) Name() string {
	return "sinusoidal"
}

func (s *sinusoidal) Generate() {
	var t time.Time
	startAt := s.env.CurrentMovementTime()
	twoPi := 2.0 * math.Pi
	for t = startAt; t.Before(s.env.HaltTime()); t = t.Add(1 * time.Second) {
		ampl := float64(s.amplitude)
		perd := float64(s.period.Seconds())
		tsec := float64(t.Unix())

		rps := ampl*math.Sin(twoPi*(tsec/perd)) + ampl
		roundedRPS := int(math.Round(rps))

		uniRand := NewUniformRandom(s.env, s.source, s.buffer, UniformConfig{
			NumberOfRequests: roundedRPS,
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
	}
}

func NewSinusoidal(env simulator.Environment, source model.TrafficSource, buffer model.RequestsRoutingStock, config SinusoidalConfig) Pattern {
	return &sinusoidal{
		env:       env,
		amplitude: config.Amplitude,
		period:    config.Period,
		source:    source,
		buffer:    buffer,
	}
}
