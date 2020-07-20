package skplug

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
)

var _ Plugin = &GRPCClient{}

// GRPCClient is an implementation of Plugin that talks over RPC.
type GRPCClient struct {
	broker *plugin.GRPCBroker
	client proto.PluginClient
}

type Autoscaler proto.Autoscaler
type Pod proto.Pod
type Object interface {
	isObject()
}

func (o *Autoscaler) isObject() {}
func (o *Pod) isObject()        {}

var _ Object = &Autoscaler{}
var _ Object = &Pod{}

func (m *GRPCClient) Event(partition string, time int64, typ proto.EventType, object Object) error {
	req := &proto.EventRequest{
		Partition: partition,
		Time:      time,
		Type:      typ,
	}
	switch v := object.(type) {
	case *Autoscaler:
		req.ObjectOneof = &proto.EventRequest_Autoscaler{(*proto.Autoscaler)(v)}
	case *Pod:
		req.ObjectOneof = &proto.EventRequest_Pod{(*proto.Pod)(v)}
	default:
		return fmt.Errorf("unknown type: %T", object)
	}
	_, err := m.client.Event(context.Background(), req)
	return err
}

func (m *GRPCClient) Stat(partition string, stats []*proto.Stat) error {
	_, err := m.client.Stat(context.Background(), &proto.StatRequest{
		Partition: partition,
		Stat:      stats,
	})
	return err
}

func (m *GRPCClient) Scale(partition string, time int64) (rec int32, err error) {
	resp, err := m.client.Scale(context.Background(), &proto.ScaleRequest{
		Partition: partition,
		Time:      time,
	})
	if err != nil {
		return 0, err
	}
	return resp.Rec, nil
}

var _ proto.PluginServer = &GRPCServer{}

// GRPCServer is the gRPC server that the GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl Plugin

	broker *plugin.GRPCBroker
}

func (m *GRPCServer) Event(ctx context.Context, req *proto.EventRequest) (*proto.Empty, error) {
	var o Object
	switch v := req.ObjectOneof.(type) {
	case *proto.EventRequest_Autoscaler:
		o = (*Autoscaler)(v.Autoscaler)
	case *proto.EventRequest_Pod:
		o = (*Pod)(v.Pod)
	default:
		return nil, fmt.Errorf("unknown type: %T", req.ObjectOneof)
	}
	err := m.Impl.Event(req.Partition, req.Time, req.Type, o)
	if err != nil {
		return nil, err
	}
	return &proto.Empty{}, nil
}

func (m *GRPCServer) Stat(ctx context.Context, req *proto.StatRequest) (*proto.Empty, error) {
	err := m.Impl.Stat(req.Partition, req.Stat)
	if err != nil {
		return nil, err
	}
	return &proto.Empty{}, nil
}

func (m *GRPCServer) Scale(ctx context.Context, req *proto.ScaleRequest) (*proto.ScaleResponse, error) {
	rec, err := m.Impl.Scale(req.Partition, req.Time)
	if err != nil {
		return nil, err
	}
	return &proto.ScaleResponse{
		Rec: rec,
	}, nil
}
