package model

import "skenario/pkg/simulator"

type MetricsSourceStock interface {
	simulator.SourceStock
}

type metricsSourceStock struct {
	env           simulator.Environment
	replicaEntity ReplicaEntity
}

func (mss *metricsSourceStock) Name() simulator.StockName {
	return "MetricsSource"
}

func (mss *metricsSourceStock) KindStocked() simulator.EntityKind {
	return "Metrics"
}

func (mss *metricsSourceStock) Count() uint64 {
	return 0
}

func (mss *metricsSourceStock) EntitiesInStock() []*simulator.Entity {
	return []*simulator.Entity{}
}

func (mss *metricsSourceStock) Remove(entity *simulator.Entity) simulator.Entity {
	return NewMetricsEntity(mss.env, mss.replicaEntity.Stats())
}

func NewMetricsSourceStock(env simulator.Environment, replicaEntity ReplicaEntity) MetricsSourceStock {
	return &metricsSourceStock{
		env:           env,
		replicaEntity: replicaEntity,
	}
}
