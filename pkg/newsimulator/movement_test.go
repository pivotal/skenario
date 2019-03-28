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
		movement = NewMovement("test movement kind", theTime, fromStock, toStock, "test note")
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
		it("has notes", func() {
			assert.Equal(t, movement.Notes(), []string{"test note"})
		})
	})

	describe("AddNote()", func() {
		it("adds a note", func() {
			movement.AddNote("added note")
			assert.Equal(t, movement.Notes()[1], "added note")
		})
	})
}
