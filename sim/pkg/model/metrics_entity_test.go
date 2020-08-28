package model

import (
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestMetricsEntity(t *testing.T) {
	spec.Run(t, "Metrics Entity", testMetricsEntity, spec.Report(report.Terminal{}))
}

func testMetricsEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject MetricsEntity
	var stats = []*proto.Stat{}

	it.Before(func() {
		subject = NewMetricsEntity(stats)
		assert.NotNil(t, subject)
	})

	describe("NewMetricsEntity()", func() {
		it("sets stat", func() {
			assert.Equal(t, stats, subject.GetStats())
		})
	})

	describe("Entity interface", func() {
		it("Name() creates sequential names", func() {
			beforeName := subject.Name()
			subject = NewMetricsEntity(stats)
			afterName := subject.Name()
			assert.NotEqual(t, beforeName, afterName)
		})

		it("implements Kind()", func() {
			assert.Equal(t, simulator.EntityKind("Metrics"), subject.Kind())
		})
	})
}
