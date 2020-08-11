package main

import (
	"fmt"
	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
)

const (
	pluginType = "fake-plugin"
)

type partition string
type pod_name string

var _ skplug.Plugin = &fakePluginServer{}

type fakePluginServer struct {
}

func newPluginServer() *fakePluginServer {
	return &fakePluginServer{}
}

func (p *fakePluginServer) Event(part string, time int64, typ proto.EventType, object skplug.Object) error {
	switch o := object.(type) {
	case *skplug.Autoscaler:
		switch typ {
		case proto.EventType_CREATE:
			return p.createAutoscaler(partition(part), o)
		case proto.EventType_UPDATE:
			return fmt.Errorf("update autoscaler event not supported")
		case proto.EventType_DELETE:
			return p.deleteAutoscaler(partition(part))
		default:
			return fmt.Errorf("unhandled event type: %v for object type: %T", typ, object)
		}
	case *skplug.Pod:
		switch typ {
		case proto.EventType_CREATE:
			return p.createPod(partition(part), o)
		case proto.EventType_UPDATE:
			return p.updatePod(partition(part), o)
		case proto.EventType_DELETE:
			return p.deletePod(partition(part), o)
		default:
			return fmt.Errorf("unhandled event type: %v for object type: %T", typ, object)
		}
	default:
		return fmt.Errorf("unhandled object type: %T", object)
	}
}

func (fp *fakePluginServer) Stat(part string, stat []*proto.Stat) error {
	return nil
}

func (fp *fakePluginServer) HorizontalRecommendation(part string, time int64) (rec int32, err error) {
	switch part {
	case "1":
		return 1, nil
	case "2":
		return 2, nil
	default:
		return 0, nil
	}
}

func (fp *fakePluginServer) VerticalRecommendation(part string, time int64) (rec []*proto.RecommendedPodResources, err error) {
	return []*proto.RecommendedPodResources{
		{
			PodName:    "Pod1",
			LowerBound: 1,
			UpperBound: 100,
			Target:     50,
		},
	}, nil
}

func (fp *fakePluginServer) createAutoscaler(part partition, a *skplug.Autoscaler) error {
	return nil
}

func (fp *fakePluginServer) deleteAutoscaler(part partition) error {
	return nil
}

func (fp *fakePluginServer) createPod(part partition, pod *skplug.Pod) error {
	return nil
}

func (fp *fakePluginServer) updatePod(part partition, pod *skplug.Pod) error {
	return nil
}

func (fp *fakePluginServer) deletePod(part partition, pod *skplug.Pod) error {
	return nil
}

func (fp *fakePluginServer) GetCapabilities() (rec []proto.Capability, err error) {
	return []proto.Capability{proto.Capability_EVENT, proto.Capability_STAT, proto.Capability_HORIZONTAL_RECOMMENDATION, proto.Capability_VERTICAL_RECOMMENDATION}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: skplug.Handshake,
		Plugins: map[string]plugin.Plugin{
			"autoscaler": &skplug.AutoscalerPlugin{Impl: newPluginServer()},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
