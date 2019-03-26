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
