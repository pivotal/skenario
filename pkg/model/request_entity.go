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

type Request interface {
}

type RequestEntity interface {
	simulator.Entity
	Request
}

type requestEntity struct {
	env                                  simulator.Environment
	number                               int
	requestConfig                        RequestConfig
	routingStock                         RequestsRoutingStock
	utilizationForRequestMillisPerSecond *float64

	startTime *time.Time
}

var reqNumber int

func (re *requestEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("request-%d", re.number))
}

func (re *requestEntity) Kind() simulator.EntityKind {
	return "Request"
}

func NewRequestEntity(env simulator.Environment, routingStock RequestsRoutingStock, requestConfig RequestConfig) RequestEntity {
	reqNumber++
	utilizationForRequest := 0.0
	return &requestEntity{
		env:                                  env,
		number:                               reqNumber,
		routingStock:                         routingStock,
		requestConfig:                        requestConfig,
		utilizationForRequestMillisPerSecond: &utilizationForRequest,
	}
}
