package model

import (
	"fmt"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"skenario/pkg/simulator"
)

type Metrics interface {
	GetStats() []*proto.Stat
}

type MetricsEntity interface {
	simulator.Entity
	Metrics
}

type metricsEntity struct {
	number int
	stats  []*proto.Stat
}

var metricsNum int

func (me *metricsEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("metrics-%d", me.number))
}

func (me *metricsEntity) Kind() simulator.EntityKind {
	return "Metrics"
}

func (me *metricsEntity) GetStats() []*proto.Stat {
	return me.stats
}

func NewMetricsEntity(stats []*proto.Stat) MetricsEntity {
	metricsNum++
	return &metricsEntity{
		number: metricsNum,
		stats:  stats,
	}
}
