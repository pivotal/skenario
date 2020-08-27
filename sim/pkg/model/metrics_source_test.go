package model

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestMetricsSourceStock(t *testing.T) {
	spec.Run(t, "Metrics Source Stock", testMetricsSourceStock, spec.Report(report.Terminal{}))
}

func testMetricsSourceStock(t *testing.T, describe spec.G, it spec.S) {
	var subject MetricsSourceStock
	var rawSubject *metricsSourceStock
	var envFake *FakeEnvironment
	var replica ReplicaEntity

	it.Before(func() {
		envFake = NewFakeEnvironment()
		failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
		replica = NewReplicaEntity(envFake, &failedSink)
		subject = NewMetricsSourceStock(envFake, replica)
		rawSubject = subject.(*metricsSourceStock)
	})

	describe("NewMetricsSourceStock", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})
		it("sets a replica", func() {
			assert.Equal(t, replica, rawSubject.replicaEntity)
		})
	})

	describe("Name()", func() {
		it("is called MetricsSource", func() {
			assert.Equal(t, simulator.StockName("MetricsSource"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Metrics", func() {
			assert.Equal(t, simulator.EntityKind("Metrics"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("gives 0", func() {
			assert.Zero(t, subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("always empty", func() {
			assert.Equal(t, []*simulator.Entity{}, subject.EntitiesInStock())
		})
	})

	describe("Remove()", func() {
		var entity1, entity2 simulator.Entity

		it.Before(func() {
			entity1 = subject.Remove(nil)
			assert.NotNil(t, entity1)
			entity2 = subject.Remove(nil)
			assert.NotNil(t, entity2)
		})

		it("creates a new MetricsEntity of EntityKind 'Metrics'", func() {
			assert.IsType(t, &metricsEntity{}, entity1)
			assert.Equal(t, simulator.EntityKind("Metrics"), entity1.Kind())
		})
	})
}
