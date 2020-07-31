package plugin

import (
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
)

type Capability string

const (
	Capability_EVENT              Capability = "event"
	Capability_STAT               Capability = "stat"
	Capability_SCALE_VERTICALLY   Capability = "scale_vertically"
	Capability_SCALE_HORIZONTALLY Capability = "scale_horizontally"
)

type PluginPartition interface {
	Event(time int64, typ proto.EventType, object skplug.Object) error
	Stat(stat []*proto.Stat) error
	ScaleHorizontally(time int64) (rec int32, err error)
	ScaleVertically(time int64) (rec []*proto.RecommendedPodResources, err error)
	GetCapabilities() []Capability
}
