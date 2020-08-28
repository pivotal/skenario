package model

import (
	"skenario/pkg/simulator"
	"time"
)

type MetricsPipelineStock interface {
	simulator.ThroughStock
}

type metricsPipelineStock struct {
	env      simulator.Environment
	pipeline simulator.ThroughStock
	sink     MetricsSinkStock
}

var metricsLagDuration = 4 * time.Second

func (mpls *metricsPipelineStock) Name() simulator.StockName {
	return mpls.pipeline.Name()
}

func (mpls *metricsPipelineStock) KindStocked() simulator.EntityKind {
	return mpls.pipeline.KindStocked()
}

func (mpls *metricsPipelineStock) Count() uint64 {
	return mpls.pipeline.Count()
}

func (mpls *metricsPipelineStock) EntitiesInStock() []*simulator.Entity {
	return mpls.pipeline.EntitiesInStock()
}

func (mpls *metricsPipelineStock) Add(entity simulator.Entity) error {
	err := mpls.pipeline.Add(entity)
	if err != nil {
		return err
	}
	//get metrics and pass it to sink with a delay
	mpls.env.AddToSchedule(simulator.NewMovement(
		"send_metrics_to_sink",
		mpls.env.CurrentMovementTime().Add(metricsLagDuration),
		mpls.pipeline,
		mpls.sink,
		&entity,
	))
	return nil
}

func (mpls *metricsPipelineStock) Remove(entity *simulator.Entity) simulator.Entity {
	return mpls.pipeline.Remove(entity)
}

func NewMetricsPipeLineStock(env simulator.Environment) MetricsPipelineStock {
	return &metricsPipelineStock{
		env:      env,
		pipeline: simulator.NewArrayThroughStock("MetricsPipeline", "Metrics"),
		sink:     NewMetricsSinkStock(env),
	}
}
