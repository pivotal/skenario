package plugin

import (
	"fmt"
	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
)

type hpaPluginPartition struct {
	partition string
}

var hpaPartitionSequence int32 = 0
var hpaPluginServer skplug.Plugin
var hpaClient *plugin.Client

func NewHpaPluginPartition() PluginPartition {
	return &hpaPluginPartition{
		partition: strconv.Itoa(int(atomic.AddInt32(&hpaPartitionSequence, 1))),
	}
}

func InitHpaPlugin() {
	// We don't want to see the plugin logs.
	//log.SetOutput(ioutil.Discard)

	// We're a host. Start by launching the plugin process.
	hpaClient = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: skplug.Handshake,
		Plugins:         skplug.PluginMap,
		Cmd:             exec.Command("sh", "-c", os.Getenv("SKENARIO_PLUGIN")),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
	})

	// Connect via RPC
	rpcClient, err := hpaClient.Client()
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

	// We should have a PluginDispatcher now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	hpaPluginServer = raw.(skplug.Plugin)
}

func ShutdownHpaPlugin() {
	hpaClient.Kill()
}
func (p *hpaPluginPartition) Event(time int64, typ proto.EventType, object skplug.Object) error {
	return hpaPluginServer.Event(p.partition, time, typ, object)
}

func (p *hpaPluginPartition) Stat(stat []*proto.Stat) error {
	return hpaPluginServer.Stat(p.partition, stat)
}

func (p *hpaPluginPartition) ScaleHorizontally(time int64) (rec int32, err error) {
	return hpaPluginServer.HorizontalRecommendation(p.partition, time)
}
func (p *hpaPluginPartition) ScaleVertically(time int64) (rec []*proto.RecommendedPodResources, err error) {
	panic("unimplemented")
}
func (p *hpaPluginPartition) GetCapabilities() []Capability {
	return []Capability{Capability_EVENT, Capability_STAT, Capability_SCALE_HORIZONTALLY}
}
