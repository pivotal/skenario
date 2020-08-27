package model

import "skenario/pkg/simulator"

type MetricsSinkStock interface {
	simulator.SinkStock
}

type metricsSinkStock struct {
	env  simulator.Environment
	sink simulator.SinkStock
}

func (mss *metricsSinkStock) Name() simulator.StockName {
	return mss.sink.Name()
}

func (mss *metricsSinkStock) KindStocked() simulator.EntityKind {
	return mss.sink.KindStocked()
}

func (mss *metricsSinkStock) Count() uint64 {
	return mss.sink.Count()
}

func (mss *metricsSinkStock) EntitiesInStock() []*simulator.Entity {
	return mss.sink.EntitiesInStock()
}

func (mss *metricsSinkStock) Add(entity simulator.Entity) error {
	err := mss.sink.Add(entity)
	if err != nil {
		return err
	}
	metrics := entity.(MetricsEntity)

	//pass stats to autoscalers
	err = mss.env.Plugin().Stat(metrics.GetStats())

	if err != nil {
		panic(err)
	}
	return nil
}

func NewMetricsSinkStock(env simulator.Environment) MetricsSinkStock {
	return &metricsSinkStock{
		env:  env,
		sink: simulator.NewSinkStock("MetricsSink", "Metrics"),
	}
}
