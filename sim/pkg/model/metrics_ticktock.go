package model

import (
	"fmt"
	"skenario/pkg/simulator"
	"time"
)

type MetricsTicktockStock interface {
	simulator.ThroughStock
}

type metricsTicktockStock struct {
	env             simulator.Environment
	replicaEntity   ReplicaEntity
	metricsSource   simulator.SourceStock
	metricsPipeline MetricsPipelineStock
}

var metricsTickInterval = 10 * time.Second

func (mts *metricsTicktockStock) Name() simulator.StockName {
	return "Metrics Ticktock"
}

func (mts *metricsTicktockStock) KindStocked() simulator.EntityKind {
	return "Replica"
}

func (mts *metricsTicktockStock) Count() uint64 {
	return 1
}

func (mts *metricsTicktockStock) EntitiesInStock() []*simulator.Entity {
	entity := mts.replicaEntity.(simulator.Entity)
	return []*simulator.Entity{&entity}
}

func (mts *metricsTicktockStock) Add(entity simulator.Entity) error {
	if mts.replicaEntity != entity {
		return fmt.Errorf("'%+v' is different from the entity given at creation time, '%+v'", entity, mts.replicaEntity)
	}

	mts.env.AddToSchedule(simulator.NewMovement(
		"send_metrics_to_pipeline",
		mts.env.CurrentMovementTime().Add(1*time.Nanosecond),
		mts.metricsSource,
		mts.metricsPipeline,
		nil,
	))

	mts.env.AddToSchedule(simulator.NewMovement(
		"metrics_tick",
		mts.env.CurrentMovementTime().Add(metricsTickInterval),
		mts,
		mts,
		&entity,
	))
	return nil
}
func (mts *metricsTicktockStock) Remove(entity *simulator.Entity) simulator.Entity {
	return mts.replicaEntity
}

func NewMetricsTickTockStock(env simulator.Environment, replicaEntity ReplicaEntity) MetricsTicktockStock {
	return &metricsTicktockStock{
		env:             env,
		replicaEntity:   replicaEntity,
		metricsSource:   NewMetricsSourceStock(env, replicaEntity),
		metricsPipeline: NewMetricsPipeLineStock(env),
	}
}
