package plugin

import (
	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"strconv"
	"sync/atomic"
)

type vpaPluginPartition struct {
	partition string
}

var vpaPartitionSequence int32 = 0
var vpaPluginServer skplug.Plugin
var vpaClient *plugin.Client

func NewVpaPluginPartition() PluginPartition {
	return &vpaPluginPartition{
		partition: strconv.Itoa(int(atomic.AddInt32(&vpaPartitionSequence, 1))),
	}
}

func InitVpaPlugin() {
	//TODO implement when VPA plugin is ready
}

func ShutdownVpaPlugin() {
	//TODO implement when VPA plugin is ready
}

func (p *vpaPluginPartition) Event(time int64, typ proto.EventType, object skplug.Object) error {
	return vpaPluginServer.Event(p.partition, time, typ, object)
}

func (p *vpaPluginPartition) Stat(stat []*proto.Stat) error {
	return vpaPluginServer.Stat(p.partition, stat)
}

func (p *vpaPluginPartition) ScaleHorizontally(time int64) (rec int32, err error) {
	panic("unimplemented")
}
func (p *vpaPluginPartition) ScaleVertically(time int64) (rec []*proto.RecommendedPodResources, err error) {
	return vpaPluginServer.VerticalRecommendation(p.partition, time)
}
func (p *vpaPluginPartition) GetCapabilities() []Capability {
	return []Capability{Capability_EVENT, Capability_STAT, Capability_SCALE_VERTICALLY}
}
