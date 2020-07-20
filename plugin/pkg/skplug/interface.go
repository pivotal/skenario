package skplug

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SKENARIO_PLUGIN",
	MagicCookieValue: "skplug",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"autoscaler": &AutoscalerPlugin{},
}

// Plugin is the interface that we're exposing as a plugin.
type Plugin interface {
	Event(partition string, time int64, typ proto.EventType, object Object) error
	Stat(partition string, stat []*proto.Stat) error
	Scale(partition string, time int64) (rec int32, err error)
}

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type AutoscalerPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Plugin
}

func (p *AutoscalerPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServer(s, &GRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *AutoscalerPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: proto.NewPluginClient(c),
		broker: broker,
	}, nil
}

var _ plugin.GRPCPlugin = &AutoscalerPlugin{}
