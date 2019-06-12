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
	"skenario/pkg/simulator"
	"time"
)

const backoffMultiplier float64 = 1.3

type Request interface {
	NextBackoff() (backoff time.Duration, outOfAttempts bool)
}

type RequestEntity interface {
	simulator.Entity
	Request
}

type requestEntity struct {
	env         simulator.Environment
	number      int
	bufferStock RequestsBufferedStock
	nextBackoff time.Duration
	attempts    int

	// CPU seconds required to complete the request.
	cpuSecondsRequired time.Duration
	cpuSecondsConsumed time.Duration
}

var reqNumber int

func (re *requestEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("request-%d", re.number))
}

func (re *requestEntity) Kind() simulator.EntityKind {
	return "Request"
}

func (re *requestEntity) NextBackoff() (backoff time.Duration, outOfAttempts bool) {
	if re.attempts < 18 {
		re.attempts++
	} else {
		return re.nextBackoff, true
	}

	thisBackoff := re.nextBackoff
	re.nextBackoff = time.Duration(int64(float64(re.nextBackoff) * backoffMultiplier))

	return thisBackoff, outOfAttempts
}

func NewRequestEntity(env simulator.Environment, buffer RequestsBufferedStock) RequestEntity {
	reqNumber++
	return &requestEntity{
		env:                env,
		number:             reqNumber,
		bufferStock:        buffer,
		nextBackoff:        100 * time.Millisecond,
		cpuSecondsRequired: 500 * time.Millisecond,
	}
}
