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
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestMovement(t *testing.T) {
	spec.Run(t, "Movement spec", testMovement, spec.Report(report.Terminal{}))
}

func testMovement(t *testing.T, describe spec.G, it spec.S) {
	var fromStock SourceStock
	var toStock SinkStock
	var movement Movement
	theTime := time.Now()

	it.Before(func() {
		fromStock = NewSourceStock("test from source", "test entity kind")
		toStock = NewSinkStock("test from source", "test entity kind")
		movement = NewMovement("test movement kind", theTime, fromStock, toStock)
	})

	describe("Kind()", func() {
		it("has a kind", func() {
			assert.Equal(t, MovementKind("test movement kind"), movement.Kind())
		})
	})

	describe("OccursAt()", func() {
		it("has a time", func() {
			assert.Equal(t, movement.OccursAt(), theTime)
		})
	})

	describe("From()", func() {
		it("has a Source stock", func() {
			assert.Equal(t, movement.From(), fromStock)
		})
	})

	describe("To()", func() {
		it("has a Sink stock", func() {
			assert.Equal(t, movement.To(), toStock)
		})
	})

	describe("Notes()", func() {
		it("starts without any notes", func() {
			assert.Equal(t, movement.Notes(), []string{})
		})
	})

	describe("AddNote()", func() {
		it("adds a note", func() {
			movement.AddNote("added note")
			assert.Equal(t, movement.Notes()[0], "added note")
		})
	})
}
