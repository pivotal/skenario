package model

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestMetricsPipelineStock(t *testing.T) {
	spec.Run(t, "Metrics Pipeline Stock", testMetricsPipelineStock, spec.Report(report.Terminal{}))
}

func testMetricsPipelineStock(t *testing.T, describe spec.G, it spec.S) {
	var subject MetricsPipelineStock
	var rawSubject *metricsPipelineStock
	var envFake *FakeEnvironment
	var metrics MetricsEntity

	it.Before(func() {
		envFake = NewFakeEnvironment()
		failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
		replica := NewReplicaEntity(envFake, &failedSink)
		metrics = NewMetricsEntity(replica.Stats())
		subject = NewMetricsPipeLineStock(envFake)
		rawSubject = subject.(*metricsPipelineStock)
	})

	describe("NewMetricsPipeLineStock", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})

		it("creates NewMetricsSinkStock", func() {
			assert.IsType(t, &metricsSinkStock{}, rawSubject.sink)
		})

		it("creates a pipeline ThroughStock", func() {
			assert.NotNil(t, rawSubject.pipeline)
			assert.Equal(t, simulator.StockName("MetricsPipeline"), rawSubject.pipeline.Name())
			assert.Equal(t, simulator.EntityKind("Metrics"), rawSubject.pipeline.KindStocked())
		})
	})

	describe("Name()", func() {
		it("is called MetricsPipeline", func() {
			assert.Equal(t, simulator.StockName("MetricsPipeline"), subject.Name())
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

		it("scheduled a movement send_metrics_to_sink", func() {
			assert.Equal(t, envFake.Movements[0].Kind(), simulator.MovementKind("send_metrics_to_sink"))
			assert.IsType(t, &metricsSinkStock{}, envFake.Movements[0].To())
		})
	})

	describe("Remove()", func() {
		it.Before(func() {
			err := subject.Add(metrics)
			assert.Nil(t, err)
		})
		it("revomes exactly what we put there", func() {
			assert.Equal(t, subject.Remove(nil), metrics)
		})
	})
}
