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
	"math/rand"
	"time"

	"knative-simulator/pkg/simulator"
)

type Traffic struct {
	env       *simulator.Environment
	endpoints *ReplicaEndpoints
	buffer    *KBuffer
	beginTime time.Time
	endTime   time.Time
}

func NewTraffic(env *simulator.Environment, buffer *KBuffer, endpoints *ReplicaEndpoints, begin time.Time, runFor time.Duration) *Traffic {
	return &Traffic{
		env:       env,
		endpoints: endpoints,
		buffer:    buffer,
		beginTime: begin,
		endTime:   begin.Add(runFor),
	}
}

func (tr *Traffic) Identity() simulator.ProcessIdentity {
	return "Traffic"
}

func (tr *Traffic) UpdateStock(movement simulator.StockMovementEvent) {
	// do nothing, this is to match the Stock type
}

func (tr *Traffic) Run() {
	t := tr.beginTime

	for {
		r := rand.Int63n(100000)
		t = t.Add(time.Duration(r) * time.Millisecond)

		if t.After(tr.endTime) {
			return
		}

		req := NewRequest(tr.env, tr.buffer, t)
		tr.env.Schedule(simulator.NewMovementEvent(
			bufferRequest,
			t,
			req,
			tr,
			tr.buffer,
		))
	}
}
