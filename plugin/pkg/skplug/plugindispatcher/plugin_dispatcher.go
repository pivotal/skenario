package plugindispatcher

import (
	"fmt"
	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"os"
	"os/exec"
)

var capabilityToPlugins map[proto.Capability][]*skplug.Plugin
var pluginsServers []skplug.Plugin
var pluginsClients []*plugin.Client

func Event(partition string, time int64, typ proto.EventType, object skplug.Object) error {
	for _, pluginServer := range capabilityToPlugins[proto.Capability_EVENT] {
		err := (*pluginServer).Event(partition, time, typ, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func Stat(partition string, stat []*proto.Stat) error {
	for _, pluginServer := range capabilityToPlugins[proto.Capability_STAT] {
		err := (*pluginServer).Stat(partition, stat)
		if err != nil {
			return err
		}
	}
	return nil
}

func HorizontalRecommendation(partition string, time int64) (rec int32, err error) {
	for _, pluginServer := range capabilityToPlugins[proto.Capability_HORIZONTAL_RECOMMENDATION] {
		return (*pluginServer).HorizontalRecommendation(partition, time)
	}
	return 0, nil
}

func VerticalRecommendation(partition string, time int64) (rec []*proto.RecommendedPodResources, err error) {
	for _, pluginServer := range capabilityToPlugins[proto.Capability_VERTICAL_RECOMMENDATION] {
		return (*pluginServer).VerticalRecommendation(partition, time)
	}
	return []*proto.RecommendedPodResources{}, nil
}

func Init(pluginsPaths []string) {
	capabilityToPlugins = make(map[proto.Capability][]*skplug.Plugin, 0)
	// We don't want to see the plugin logs.
	//log.SetOutput(ioutil.Discard)
	for _, pluginPath := range pluginsPaths {
		// We're a host. Start by launching the plugin process.
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: skplug.Handshake,
			Plugins:         skplug.PluginMap,
			Cmd:             exec.Command("sh", "-c", pluginPath),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		})

		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			fmt.Println("Error:", err.Error())
			os.Exit(1)
		}

		// Request the plugin
		raw, err := rpcClient.Dispense("autoscaler")
		if err != nil {
			fmt.Println("Error:", err.Error())
			os.Exit(1)
		}

		// We should have a Plugin now! This feels like a normal interface
		// implementation but is in fact over an RPC connection.
		pluginServer := raw.(skplug.Plugin)
		registerPlugin(&pluginServer)
		pluginsServers = append(pluginsServers, pluginServer)
		pluginsClients = append(pluginsClients, client)
	}
}

func registerPlugin(pluginServer *skplug.Plugin) {
	capabilities, _ := (*pluginServer).GetCapabilities()
	for _, capability := range capabilities {
		capabilityToPlugins[capability] = append(capabilityToPlugins[capability], pluginServer)
	}

	if len(capabilityToPlugins[proto.Capability_HORIZONTAL_RECOMMENDATION]) > 1 {
		panic("Plugin Dispatcher doesn't support more that one plugin with horizontal scaling simultaneously")
	}
	if len(capabilityToPlugins[proto.Capability_VERTICAL_RECOMMENDATION]) > 1 {
		panic("Plugin Dispatcher doesn't support more that one plugin with vertical scaling simultaneously")
	}
}

func Shutdown() {
	for _, client := range pluginsClients {
		client.Kill()
	}
}
