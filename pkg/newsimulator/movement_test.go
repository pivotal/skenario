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
		movement = NewMovement(theTime, fromStock, toStock, "test note")
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

	describe("OccursAt()", func() {
		it("has a time", func() {
			assert.Equal(t, movement.OccursAt(), theTime)
		})
	})

	describe("Note()", func() {
		it("has a note", func() {
			assert.Equal(t, movement.Note(), "test note")
		})
	})
}
