package plugindispatcher

import (
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"skenario/pkg/plugin"
)

type PluginDispatcher interface {
	Event(time int64, typ proto.EventType, object skplug.Object) error
	Stat(stat []*proto.Stat) error
	ScaleHorizontally(time int64) (rec int32, err error)
	ScaleVertically(time int64) (rec []*proto.RecommendedPodResources, err error)
}

type pluginDispatcher struct {
	capabilityToPlugins map[plugin.Capability][]*plugin.PluginPartition
}

func (pd *pluginDispatcher) Event(time int64, typ proto.EventType, object skplug.Object) error {
	for _, pluginPartition := range pd.capabilityToPlugins[plugin.Capability_EVENT] {
		err := (*pluginPartition).Event(time, typ, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pd *pluginDispatcher) Stat(stat []*proto.Stat) error {
	for _, pluginPartition := range pd.capabilityToPlugins[plugin.Capability_STAT] {
		err := (*pluginPartition).Stat(stat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pd *pluginDispatcher) ScaleHorizontally(time int64) (rec int32, err error) {
	for _, pluginPartition := range pd.capabilityToPlugins[plugin.Capability_SCALE_HORIZONTALLY] {
		return (*pluginPartition).ScaleHorizontally(time)
	}
	return 0, nil
}

func (pd *pluginDispatcher) ScaleVertically(time int64) (rec []*proto.RecommendedPodResources, err error) {
	for _, pluginPartition := range pd.capabilityToPlugins[plugin.Capability_SCALE_VERTICALLY] {
		return (*pluginPartition).ScaleVertically(time)
	}
	return []*proto.RecommendedPodResources{}, nil
}

func (pd *pluginDispatcher) addPlugin(pluginPartition plugin.PluginPartition) {
	for _, capability := range pluginPartition.GetCapabilities() {
		pd.capabilityToPlugins[capability] = append(pd.capabilityToPlugins[capability], &pluginPartition)
	}

	if len(pd.capabilityToPlugins[plugin.Capability_SCALE_HORIZONTALLY]) > 1 {
		panic("Plugin Dispatcher doesn't support more that one plugin with horizontal scaling simultaneously")
	}
	if len(pd.capabilityToPlugins[plugin.Capability_SCALE_VERTICALLY]) > 1 {
		panic("Plugin Dispatcher doesn't support more that one plugin with vertical scaling simultaneously")
	}
}

func NewPluginDispatcher() PluginDispatcher {
	pluginDispatcher := &pluginDispatcher{capabilityToPlugins: make(map[plugin.Capability][]*plugin.PluginPartition)}

	//add new plugins here
	pluginDispatcher.addPlugin(plugin.NewHpaPluginPartition())

	return pluginDispatcher
}

func Init() {
	plugin.InitHpaPlugin()
	plugin.InitVpaPlugin()
}

func Shutdown() {
	plugin.ShutdownHpaPlugin()
	plugin.ShutdownVpaPlugin()
}
