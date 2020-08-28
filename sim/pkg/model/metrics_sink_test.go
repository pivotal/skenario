package model

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestMetricsSinkStock(t *testing.T) {
	spec.Run(t, "Metrics Sink Stock", testMetricsSinkStock, spec.Report(report.Terminal{}))
}

func testMetricsSinkStock(t *testing.T, describe spec.G, it spec.S) {
	var subject MetricsSinkStock
	var rawSubject *metricsSinkStock
	var envFake *FakeEnvironment
	var metrics MetricsEntity

	it.Before(func() {
		envFake = NewFakeEnvironment()
		failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
		replica := NewReplicaEntity(envFake, &failedSink)
		metrics = NewMetricsEntity(replica.Stats())
		subject = NewMetricsSinkStock(envFake)
		rawSubject = subject.(*metricsSinkStock)
	})

	describe("NewMetricsSinkStock", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})
	})

	describe("Name()", func() {
		it("is called MetricsSink", func() {
			assert.Equal(t, simulator.StockName("MetricsSink"), subject.Name())
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
		it("gives 1", func() {
			subject.Add(metrics)
			assert.Equal(t, uint64(1), subject.Count())
		})
	})

	describe("EntitiesInStock()", func() {
		it("empty", func() {
			assert.Len(t, subject.EntitiesInStock(), 0)
		})
		it("has 1 entity", func() {
			subject.Add(metrics)
			assert.Len(t, subject.EntitiesInStock(), 1)
			assert.Equal(t, *subject.EntitiesInStock()[0], metrics)

		})
	})

	describe("Add()", func() {
		it.Before(func() {
			err := subject.Add(metrics)
			assert.Nil(t, err)
		})

		it("pass stats to plugin", func() {
			fakePlugin := envFake.ThePlugin.(*FakePluginPartition)
			assert.Equal(t, fakePlugin.stats, metrics.GetStats())
		})
	})
}
