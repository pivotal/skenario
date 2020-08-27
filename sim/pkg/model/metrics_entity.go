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
	env    simulator.Environment
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

func NewMetricsEntity(env simulator.Environment, stats []*proto.Stat) MetricsEntity {
	metricsNum++
	return &metricsEntity{
		env:    env,
		number: metricsNum,
		stats:  stats,
	}
}
