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
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestScraperTicktock(t *testing.T) {
	spec.Run(t, "Scraper Ticktock stock", testScraperTicktock, spec.Report(report.Terminal{}))
}

func testScraperTicktock(t *testing.T, describe spec.G, it spec.S) {
	var subject ScraperTicktock
	var rawSubject *scraperTicktock
	var collectorFake *FakeCollector
	var scraperFake *FakeScraper

	it.Before(func() {
		collectorFake = new(FakeCollector)
		scraperFake = new(FakeScraper)
		subject = NewScraperTicktockStock (collectorFake, scraperFake)
		rawSubject = subject.(*scraperTicktock)
	})

	describe("NewAutoscalerTicktockStock()", func() {
		it("sets the entity", func() {
			assert.Equal(t, simulator.EntityName("Scraper"), rawSubject.scraperEntity.Name())
			assert.Equal(t, simulator.EntityKind("KnativeScraper"), rawSubject.scraperEntity.Kind())
		})
	})

	describe("Name()", func() {
		it("is called 'Scraper Ticktock'", func() {
			assert.Equal(t, subject.Name(), simulator.StockName("Scraper Ticktock"))
		})
	})

	describe("KindStocked()", func() {
		it("accepts Knative Stat Scrapers", func() {
			assert.Equal(t, subject.KindStocked(), simulator.EntityKind("KnativeScraper"))
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
		it("gives back the one KnativeScraper", func() {
			assert.Equal(t, subject.Remove(), subject.Remove())
			assert.Equal(t, rawSubject.scraperEntity, subject.Remove())
		})
	})

	describe("Add()", func() {
		describe("ensuring consistency", func() {
			var differentEntity simulator.Entity

			it.Before(func() {
				differentEntity = simulator.NewEntity("Different!", "KnativeScraper")
			})

			it("returns error if the Added entity does not equal the existing entity", func() {
				err := subject.Add(differentEntity)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "different from the entity given at creation time")
			})
		})
	})
}