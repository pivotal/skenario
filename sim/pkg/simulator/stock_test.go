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

package simulator

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestStock(t *testing.T) {
	suite := spec.New("Stock suite", spec.Report(report.Terminal{}))
	suite("Stock", testStock)
	suite("SourceStock", testSourceStock)
	suite("SinkStock", testSinkStock)
	suite("ThroughStock", testThroughStock)

	suite.Run(t)
}

func testStock(t *testing.T, describe spec.G, it spec.S) {
	var subject ThroughStock
	var entity Entity

	it.Before(func() {
		subject = NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subject)

		entity = NewEntity("test entity name", "test entity kind")
		err := subject.Add(entity)
		assert.NoError(t, err)
	})

	describe("basic Stock functionality", func() {
		it("has a stock name", func() {
			assert.Equal(t, subject.Name(), StockName("test name"))
		})

		it("has a stock kind", func() {
			assert.Equal(t, subject.KindStocked(), EntityKind("test entity kind"))
		})

		it("has a stock count", func() {
			assert.Equal(t, subject.Count(), uint64(1))
		})
	})

	describe("EntitiesInStock()", func() {
		it("returns an Entity map for the current contents of the stock", func() {
			assert.Equal(t, map[Entity]bool{entity: true}, subject.EntitiesInStock())
		})
	})
}

func testSourceStock(t *testing.T, describe spec.G, it spec.S) {
	var subject SourceStock
	var subjectAsThrough ThroughStock
	var entity1, entity2 Entity

	it.Before(func() {
		subjectAsThrough = NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subjectAsThrough)

		subject = subjectAsThrough.(SourceStock)
		assert.NotNil(t, subject)

		entity1 = NewEntity("test entity 1", "test entity kind")
		err := subjectAsThrough.Add(entity1)
		assert.NoError(t, err)

		entity2 = NewEntity("test entity 2", "test entity kind")
		err = subjectAsThrough.Add(entity2)
		assert.NoError(t, err)
	})

	describe("Remove()", func() {
		it("decreases the stock count by 1", func() {
			before := subject.Count()

			subject.Remove(nil)

			after := subject.Count()
			assert.Equal(t, before-1, after)
		})

		it("removes in FIFO order", func() {
			assert.Equal(t, entity1, subject.Remove(nil))
			assert.Equal(t, entity2, subject.Remove(nil))
		})
	})
}

func testSinkStock(t *testing.T, describe spec.G, it spec.S) {
	var subject SinkStock
	var entity Entity

	it.Before(func() {
		subject = NewSinkStock("test name", "test entity kind")
		assert.NotNil(t, subject)

		entity = NewEntity("test entity name", "test entity kind")
	})

	describe("Add()", func() {
		it("adds a stock item", func() {
			err := subject.Add(entity)
			assert.NoError(t, err)
		})

		it("increases the stock count by 1", func() {
			before := subject.Count()

			err := subject.Add(entity)
			assert.NoError(t, err)

			after := subject.Count()

			assert.Equal(t, before+1, after)
		})

		describe("Entity with mismatched kind added", func() {
			it("rejects the entity", func() {
				wrongEntity := NewEntity("will explode", "wrong kind")

				err := subject.Add(wrongEntity)
				assert.Error(t, err)
			})
		})

		describe("A nil Entity is added", func() {
			it("rejects the entity", func() {
				err := subject.Add(nil)
				assert.Errorf(t, err, "was nil")
			})
		})
	})
}

func testThroughStock(t *testing.T, describe spec.G, it spec.S) {
	var subject ThroughStock
	var entity Entity

	it.Before(func() {
		subject = NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subject)
	})

	describe("Add() then Remove()", func() {
		it("gets back the item was added", func() {
			entity = NewEntity("test entity name", "test entity kind")

			err := subject.Add(entity)
			assert.NoError(t, err)

			removed := subject.Remove(nil)
			assert.NotNil(t, removed)

			assert.Equal(t, entity, removed)
		})
	})

	describe("Remove() when the stock is empty", func() {
		it("returns nil", func() {
			assert.Nil(t, subject.Remove(nil))
		})
	})
}
