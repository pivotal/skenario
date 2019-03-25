package newsimulator_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
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
	var subject newsimulator.ThroughStock

	it.Before(func() {
		subject = newsimulator.NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subject)
	})

	describe("basic Stock functionality", func() {
		it("has a stock name", func() {
			assert.Equal(t, subject.Name(), newsimulator.StockName("test name"))
		})

		it("has a stock kind", func() {
			assert.Equal(t, subject.KindStocked(), newsimulator.EntityKind("test entity kind"))
		})

		it("has a stock count", func() {
			assert.Equal(t, subject.Count(), uint64(0))
		})
	})

}

func testSourceStock(t *testing.T, describe spec.G, it spec.S) {
	var subject newsimulator.SourceStock
	var subjectAsThrough newsimulator.ThroughStock
	var entity newsimulator.Entity

	it.Before(func() {
		subjectAsThrough = newsimulator.NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subjectAsThrough)

		subject = subjectAsThrough.(newsimulator.SourceStock)
		assert.NotNil(t, subject)

		entity = newsimulator.NewEntity("test entity name", "test entity kind")
		err := subjectAsThrough.Add(entity)
		assert.NoError(t, err)
	})

	describe("Remove()", func() {
		it("decreases the stock count by 1", func() {
			before := subject.Count()

			subject.Remove()

			after := subject.Count()
			assert.Equal(t, before-1, after)
		})
	})
}

func testSinkStock(t *testing.T, describe spec.G, it spec.S) {
	var subject newsimulator.SinkStock
	var entity newsimulator.Entity

	it.Before(func() {
		subject = newsimulator.NewSinkStock("test name", "test entity kind")
		assert.NotNil(t, subject)

		entity = newsimulator.NewEntity("test entity name", "test entity kind")
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

		describe("Wrong stock type added", func() {
			it("says kaboom", func() {
				wrongEntity := newsimulator.NewEntity("will explode", "wrong kind")

				err := subject.Add(wrongEntity)
				assert.Error(t, err)
			})
		})
	})
}

func testThroughStock(t *testing.T, describe spec.G, it spec.S) {
	var subject newsimulator.ThroughStock
	var entity newsimulator.Entity

	it.Before(func() {
		subject = newsimulator.NewThroughStock("test name", "test entity kind")
		assert.NotNil(t, subject)
	})

	describe("Add() then Remove()", func() {
		it("gets back the item was added", func() {
			entity = newsimulator.NewEntity("test entity name", "test entity kind")

			err := subject.Add(entity)
			assert.NoError(t, err)

			removed := subject.Remove()
			assert.NotNil(t, removed)

			assert.Equal(t, entity, removed)
		})
	})

	describe("Remove() when the stock is empty", func() {
		it("returns nil", func() {
			assert.Nil(t, subject.Remove())
		})
	})
}
