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

func TestReplicasActive(t *testing.T) {
	spec.Run(t, "Replicas Active spec", testReplicasActive, spec.Report(report.Terminal{}))
}

func testReplicasActive(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicasActiveStock
	var rawSubject *replicasActiveStock

	it.Before(func() {
		subject = NewReplicasActiveStock()
		assert.NotNil(t, subject)

		rawSubject = subject.(*replicasActiveStock)
	})

	describe("NewReplicasActiveStock()", func() {
		it("creates a delegate ThroughStock", func() {
			assert.NotNil(t, rawSubject.delegate)
			assert.Equal(t, simulator.StockName("ReplicasActive"), rawSubject.delegate.Name())
			assert.Equal(t, simulator.EntityKind("Replica"), rawSubject.delegate.KindStocked())
		})
	})

	describe("Add()", func() {
		var replicaFake *FakeReplica

		it.Before(func() {
			replicaFake = new(FakeReplica)
			subject.Add(replicaFake)
		})

		it("tells the Replica entity that it is active", func() {
			assert.True(t, replicaFake.ActivateCalled)
		})
	})

	describe("Remove()", func() {
		var replicaFake *FakeReplica

		it.Before(func() {
			replicaFake = new(FakeReplica)
			subject.Add(replicaFake)
			subject.Remove()
		})

		it("tells the Replica entity that it is terminating", func() {
			assert.True(t, replicaFake.DeactivateCalled)
		})

		it("returns nil if it is empty", func() {
			assert.Zero(t, subject.Count())
			assert.Nil(t, subject.Remove())
		})
	})
}
