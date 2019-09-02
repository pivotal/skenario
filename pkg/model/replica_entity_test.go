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
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"knative.dev/serving/pkg/autoscaler"

	"skenario/pkg/simulator"
)

func TestReplicaEntity(t *testing.T) {
	spec.Run(t, "Replica Entity", testReplicaEntity, spec.Report(report.Terminal{}))
}

func testReplicaEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicaEntity
	var rawSubject *replicaEntity
	var envFake *FakeEnvironment

	it.Before(func() {
		envFake = new(FakeEnvironment)

		subject = NewReplicaEntity(envFake, 100)
		assert.NotNil(t, subject)

		rawSubject = subject.(*replicaEntity)
	})

	describe("NewReplicaEntity()", func() {
		it("sets an Environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})


		it("sets a RequestsComplete stock", func() {
			assert.Equal(t, simulator.StockName(fmt.Sprintf("RequestsComplete [%d]", rawSubject.number)), rawSubject.requestsComplete.Name())
		})
	})

	describe("Entity interface", func() {
		it("Name() creates sequential names", func() {
			beforeName := subject.Name()
			subject = NewReplicaEntity(envFake, 100)
			afterName := subject.Name()
			assert.NotEqual(t, beforeName, afterName)
		})

		it("implements Kind()", func() {
			assert.Equal(t, simulator.EntityKind("Replica"), subject.Kind())
		})
	})

	describe("RequestsProcessing()", func() {
		it("returns the Requests Processing stock", func() {
			assert.Contains(t, subject.RequestsProcessing().Name(), "RequestsProcessing [")
			assert.Equal(t, simulator.EntityKind("Request"), subject.RequestsProcessing().KindStocked())
		})
	})

	describe("Stat()", func() {
		describe("Creating an autoscaler.Stat struct", func() {
			var request1, request2 simulator.Entity
			var stat autoscaler.Stat

			it.Before(func() {
				rawSubject = subject.(*replicaEntity)

				request1 = simulator.NewEntity("request-1", simulator.EntityKind("Request"))
				rawSubject.requestsProcessing.Add(request1)
				request2 = simulator.NewEntity("request-2", simulator.EntityKind("Request"))
				rawSubject.requestsProcessing.Add(request2)

				stat = subject.Stat()
			})

			it("sets Time to the value provided", func() {
				assert.Equal(t, envFake.TheTime, *stat.Time)
			})

			it("sets PodName to the replica's name", func() {
				assert.Equal(t, string(subject.Name()), stat.PodName)
			})

			it("sets AverageConcurrentRequests based on RequestsProcessing.Count()", func() {
				assert.Equal(t, float64(rawSubject.requestsProcessing.Count()), stat.AverageConcurrentRequests)
			})

			it("sets RequestCount based on the number of movements to RequestsProcessing", func() {
				assert.Equal(t, float64(2), stat.RequestCount)
			})
		})
	})
}
