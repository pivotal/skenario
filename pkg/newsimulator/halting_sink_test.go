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

package newsimulator

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestHaltingSink(t *testing.T) {
	spec.Run(t, "Halting Sink spec", testHaltingSink, spec.Report(report.Terminal{}))
}

func testHaltingSink(t *testing.T, describe spec.G, it spec.S) {
	var subject *haltingSink
	var mpq MovementPriorityQueue
	var e Entity

	it.Before(func() {
	 	mpq = NewMovementPriorityQueue()
		assert.False(t, mpq.IsClosed())

	 	subject = NewHaltingSink("test name", "Scenario", mpq)
		e = NewEntity("test entity", "Scenario")
	})

	describe("halting the scenario", func() {
		it("it closes the movement priority queue", func() {
			err := subject.Add(e)
			assert.NoError(t, err)
			assert.True(t, mpq.IsClosed())
		})
	})
}
