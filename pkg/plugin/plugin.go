package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"

	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
)

var pluginServer skplug.Plugin
var client *plugin.Client

func Init() {
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

func Shutdown() {
	client.Kill()
}

type PluginPartition interface {
	Event(time int64, typ proto.EventType, object skplug.Object) error
	Stat(stat []*proto.Stat) error
	Scale(time int64) (rec int32, err error)
}

type pluginPartition struct {
	partition string
}

var partitionSequence int32 = 0

func NewPluginPartition() PluginPartition {
	return &pluginPartition{
		partition: strconv.Itoa(int(atomic.AddInt32(&partitionSequence, 1))),
	}
}

func (p *pluginPartition) Event(time int64, typ proto.EventType, object skplug.Object) error {
	return pluginServer.Event(p.partition, time, typ, object)
}

func (p *pluginPartition) Stat(stat []*proto.Stat) error {
	return pluginServer.Stat(p.partition, stat)
}

func (p *pluginPartition) Scale(time int64) (rec int32, err error) {
	return pluginServer.Scale(p.partition, time)
}
