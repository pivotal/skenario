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

	"knative-simulator/pkg/simulator"
)

func TestAutoscalerTicktock(t *testing.T) {
	spec.Run(t, "Ticktock stock", testAutoscalerTicktock, spec.Report(report.Terminal{}))
}

func testAutoscalerTicktock(t *testing.T, describe spec.G, it spec.S) {
	var subject AutoscalerTicktockStock
	var rawSubject *autoscalerTicktockStock

	it.Before(func() {
		subject = NewAutoscalerTicktockStock(simulator.NewEntity("Autoscaler", "KnativeAutoscaler"))
		rawSubject = subject.(*autoscalerTicktockStock)
	})

	describe("NewAutoscalerTicktockStock()", func() {
		it("sets the entity", func() {
			assert.Equal(t, simulator.EntityName("Autoscaler"), rawSubject.autoscalerEntity.Name())
			assert.Equal(t, simulator.EntityKind("KnativeAutoscaler"), rawSubject.autoscalerEntity.Kind())
		})
	})

	describe("Name()", func() {
		it("is called 'KnativeAutoscaler Stock'", func() {
			assert.Equal(t, subject.Name(), simulator.StockName("Autoscaler Ticktock"))
		})
	})

	describe("KindStocked()", func() {
		it("accepts Knative Autoscalers", func() {
			assert.Equal(t, subject.KindStocked(), simulator.EntityKind("KnativeAutoscaler"))
		})
	})

	describe("Count()", func() {
		it("always has 1 entity stocked", func() {
			assert.Equal(t, subject.Count(), uint64(1))

			ent := subject.Remove()
			err := subject.Add(ent)
			assert.NoError(t, err)
			err = subject.Add(ent)
			assert.NoError(t, err)

			assert.Equal(t, subject.Count(), uint64(1))

			subject.Remove()
			subject.Remove()
			subject.Remove()
			assert.Equal(t, subject.Count(), uint64(1))
		})
	})

	describe("Remove()", func() {
		it("gives back the one KnativeAutoscaler", func() {
			assert.Equal(t, subject.Remove(), subject.Remove())
		})
	})

	describe("Add()", func() {
		var differentEntity simulator.Entity

		it.Before(func() {
			differentEntity = simulator.NewEntity("Different!", "KnativeAutoscaler")
		})

		it("returns error if the Added entity does not equal the existing entity", func() {
			err := subject.Add(differentEntity)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "different from the entity given at creation time")
		})
	})
}
