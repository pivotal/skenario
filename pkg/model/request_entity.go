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
	"fmt"
	"time"

	"knative-simulator/pkg/simulator"
)

const backoffMultiplier float64 = 1.3

type Request interface {
	ScheduleBackoffMovement() (outOfAttempts bool)
}

type RequestEntity interface {
	simulator.Entity
	Request
}

type requestEntity struct {
	env         simulator.Environment
	bufferStock RequestsBufferedStock
	nextBackoff time.Duration
	attempts    int
}

var reqNumber int

func (re *requestEntity) Name() simulator.EntityName {
	reqNumber++
	return simulator.EntityName(fmt.Sprintf("request-%d", reqNumber))
}

func (re *requestEntity) Kind() simulator.EntityKind {
	return "Request"
}

func (re *requestEntity) ScheduleBackoffMovement() (outOfAttempts bool) {
	if re.attempts < 18 {
		re.attempts++
	} else {
		return true
	}

	re.env.AddToSchedule(simulator.NewMovement(
		simulator.MovementKind(fmt.Sprintf("buffer_backoff_%d", re.attempts)),
		re.env.CurrentMovementTime().Add(re.nextBackoff),
		re.bufferStock,
		re.bufferStock,
	))

	re.nextBackoff = time.Duration(int64(float64(re.nextBackoff) * backoffMultiplier))
	return outOfAttempts
}

func NewRequestEntity(env simulator.Environment, buffer RequestsBufferedStock) RequestEntity {
	return &requestEntity{
		env:         env,
		bufferStock: buffer,
		nextBackoff: 100 * time.Millisecond,
	}
}
