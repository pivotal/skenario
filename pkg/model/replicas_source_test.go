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
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/simulator"
)

func TestReplicasSource(t *testing.T) {
	spec.Run(t, "Replicas Launching source", testReplicasSource, spec.Report(report.Terminal{}))
	spec.Run(t, "IPV4Sequence interface", testIPV4Sequence, spec.Report(report.Terminal{}))
}

func testReplicasSource(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicaSource
	var rawSubject *replicaSource
	var envFake *FakeEnvironment

	it.Before(func() {
		envFake = new(FakeEnvironment)

		subject = NewReplicaSource(envFake, 100)
		rawSubject = subject.(*replicaSource)
	})

	describe("NewReplicaSource", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})
	})

	describe("Name()", func() {
		it("is called ReplicaSource", func() {
			assert.Equal(t, simulator.StockName("ReplicaSource"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Replicas", func() {
			assert.Equal(t, simulator.EntityKind("Replica"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("gives 0", func() {
			assert.Equal(t, uint64(0), subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("always empty", func() {
			assert.Equal(t, []*simulator.Entity{}, subject.EntitiesInStock())
		})
	})

	describe("Remove()", func() {
		var entity1, entity2 simulator.Entity

		it.Before(func() {
			entity1 = subject.Remove()
			assert.NotNil(t, entity1)
			entity2 = subject.Remove()
			assert.NotNil(t, entity2)
		})

		it("creates a new ReplicaEntity of EntityKind 'Request'", func() {
			assert.IsType(t, &replicaEntity{}, entity1)
			assert.Equal(t, simulator.EntityKind("Replica"), entity1.Kind())
		})
	})
}

func testIPV4Sequence(t *testing.T, describe spec.G, it spec.S) {
	var rs ReplicaSource
	var subject IPV4Sequence
	var rawSubject *replicaSource
	var envFake *FakeEnvironment

	it.Before(func() {
		rs = NewReplicaSource(envFake, 100)
		subject = rs.(IPV4Sequence)
		rawSubject = rs.(*replicaSource)
	})

	describe("NextIP()", func() {
		var ipGiven string
		it.Before(func() {
			// twice to show we didn't succeed purely on init values
			ipGiven = subject.Next()
			ipGiven = subject.Next()
		})

		it("creates an IPv4 address string", func() {
			assert.Equal(t, "0.0.0.2", ipGiven)
		})

		it("increments the next IP to give out", func() {
			assert.Equal(t, uint32(3), rawSubject.nextIPValue)
		})
	})

}
