package model

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"skenario/pkg/simulator"
	"testing"
)

func TestMetricsTicktockStock(t *testing.T) {
	spec.Run(t, "Metrics Ticktock Stock", testMetricsTicktockStock, spec.Report(report.Terminal{}))
}

func testMetricsTicktockStock(t *testing.T, describe spec.G, it spec.S) {
	var subject MetricsTicktockStock
	var rawSubject *metricsTicktockStock
	var envFake *FakeEnvironment
	var replica ReplicaEntity

	it.Before(func() {
		envFake = NewFakeEnvironment()
		failedSink := simulator.NewSinkStock("fake-requestsFailed", "Request")
		replica = NewReplicaEntity(envFake, &failedSink)
		subject = NewMetricsTickTockStock(envFake, replica)
		rawSubject = subject.(*metricsTicktockStock)
	})

	describe("NewMetricsTickTockStock", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, rawSubject.env)
		})

		it("sets a replica", func() {
			assert.Equal(t, replica, rawSubject.replicaEntity)
		})
	})

	describe("Name()", func() {
		it("is called Metrics Ticktock", func() {
			assert.Equal(t, simulator.StockName("Metrics Ticktock"), subject.Name())
		})
	})

	describe("KindStocked()", func() {
		it("stocks Replica", func() {
			assert.Equal(t, simulator.EntityKind("Replica"), subject.KindStocked())
		})
	})

	describe("Count()", func() {
		it("always has 1 entity stocked", func() {
			assert.Equal(t, subject.Count(), uint64(1))

			ent := subject.Remove(nil)
			err := subject.Add(ent)
			assert.NoError(t, err)
			err = subject.Add(ent)
			assert.NoError(t, err)

			assert.Equal(t, subject.Count(), uint64(1))

			subject.Remove(nil)
			subject.Remove(nil)
			subject.Remove(nil)
			assert.Equal(t, subject.Count(), uint64(1))
		})
	})

	describe("EntitiesInStock()", func() {
		it("always has 1 entity stocked\"", func() {
			assert.Len(t, subject.EntitiesInStock(), 1)
			assert.Equal(t, *subject.EntitiesInStock()[0], replica)

		})
	})

	describe("Add()", func() {
		it.Before(func() {
			err := subject.Add(replica)
			assert.Nil(t, err)
		})

		it("scheduled a movement send_metrics_to_pipeline", func() {
			assert.Equal(t, envFake.Movements[0].Kind(), simulator.MovementKind("send_metrics_to_pipeline"))
			assert.IsType(t, &metricsSourceStock{}, envFake.Movements[0].From())
			assert.IsType(t, &metricsPipelineStock{}, envFake.Movements[0].To())
		})
		it("scheduled a movement metrics_tick", func() {
			assert.Equal(t, envFake.Movements[1].Kind(), simulator.MovementKind("metrics_tick"))
			assert.IsType(t, &metricsTicktockStock{}, envFake.Movements[1].To())
			assert.Equal(t, envFake.Movements[1].From(), envFake.Movements[1].To())
		})
	})

	describe("Remove()", func() {
		it("gives back the one Replica", func() {
			assert.Equal(t, subject.Remove(nil), subject.Remove(nil))
		})
	})
}
