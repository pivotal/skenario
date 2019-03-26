package newsimulator

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/cache"
)

func TestHaltingSink(t *testing.T) {
	spec.Run(t, "Halting Sink spec", testHaltingSink, spec.Report(report.Terminal{}))
}

func testHaltingSink(t *testing.T, describe spec.G, it spec.S) {
	var subject *haltingSink
	var heap *cache.Heap
	var e Entity

	it.Before(func() {
	 	heap = cache.NewHeap(func(obj interface{}) (s string, e error) {
			return "key", nil

		}, func(i interface{}, i2 interface{}) bool {
			return true
		})
		assert.False(t, heap.IsClosed())

	 	subject = NewHaltingSink("test name", "Scenario", heap)
		e = NewEntity("test entity", "Scenario")
	})

	describe("halting the scenario", func() {
		it("it closes the heap", func() {
			err := subject.Add(e)
			assert.NoError(t, err)
			assert.True(t, heap.IsClosed())
		})
	})
}
