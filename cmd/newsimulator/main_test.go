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

package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
)

func TestCmdMain(t *testing.T) {
	spec.Run(t, "cmd main", testMain, spec.Report(report.Terminal{}))
}

func testMain(t *testing.T, describe spec.G, it spec.S) {
	var subject Runner
	var ignoredMovement newsimulator.Movement
	var from, to newsimulator.ThroughStock

	it.Before(func() {
		subject = NewRunner()
		from = newsimulator.NewThroughStock("test from stock", "test kind")
		to = newsimulator.NewThroughStock("test to stock", "test kind")
		ignoredMovement = newsimulator.NewMovement("test movement kind", time.Now(), from, to)
		ignoredMovement.AddNote("ignored movement")

		subject.Env().AddToSchedule(ignoredMovement)
	})

	describe("RunAndReport()", func() {
		var w bytes.Buffer
		var rpt string
		var err error

		it.Before(func() {
			w = bytes.Buffer{}
			err = subject.RunAndReport(&w)
			rpt = w.String()
			assert.NoError(t, err)
		})
		it("prints completed", func() {
			assert.Contains(t, rpt, "BeforeScenario")
			assert.Contains(t, rpt, "RunningScenario")
			assert.Contains(t, rpt, "HaltedScenario")
		})

		it("prints ignored", func() {
			assert.Contains(t, rpt, "ignored movement")
		})
	})

	describe("NewRunner()", func() {
		it("has an Environment", func() {
			assert.NotNil(t, subject.Env())
		})
	})
}
