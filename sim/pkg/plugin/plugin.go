package plugin

import (
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/dispatcher"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"strconv"
	"sync/atomic"
)

type PluginPartition interface {
	Event(time int64, typ proto.EventType, object skplug.Object) error
	Stat(stat []*proto.Stat) error
	HorizontalRecommendation(time int64) (rec int32, err error)
	VerticalRecommendation(time int64) (rec []*proto.RecommendedPodResources, err error)
}

type pluginPartition struct {
	partition string
	plugin    skplug.Plugin
}

var partitionSequence int32 = 0

func NewPluginPartition(dispatcher *dispatcher.Dispatcher) PluginPartition {
	return &pluginPartition{
		partition: strconv.Itoa(int(atomic.AddInt32(&partitionSequence, 1))),
		plugin:    (*dispatcher).GetPlugin(),
	}
}

func (p *pluginPartition) Event(time int64, typ proto.EventType, object skplug.Object) error {
	return p.plugin.Event(p.partition, time, typ, object)
}

func (p *pluginPartition) Stat(stat []*proto.Stat) error {
	return p.plugin.Stat(p.partition, stat)
}

func (p *pluginPartition) HorizontalRecommendation(time int64) (rec int32, err error) {
	return p.plugin.HorizontalRecommendation(p.partition, time)
}

func (p *pluginPartition) VerticalRecommendation(time int64) (rec []*proto.RecommendedPodResources, err error) {
	return p.plugin.VerticalRecommendation(p.partition, time)
}
