package serve

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
)

var pluginServer skplug.Plugin
var client *plugin.Client

func init() {
	// We don't want to see the plugin logs.
	//log.SetOutput(ioutil.Discard)

	// We're a host. Start by launching the plugin process.
	client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: skplug.Handshake,
		Plugins:         skplug.PluginMap,
		Cmd:             exec.Command("sh", "-c", os.Getenv("SKENARIO_PLUGIN")),
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
	pluginServer = raw.(skplug.Plugin)
}

func shutdownAutoscalerPlugin() {
	client.Kill()
}
