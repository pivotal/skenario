package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/go-plugin"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"k8s.io/klog"
	"log"
	vpaplugin "plugin-k8s-vpa/pkg/plugin"
	"sync"
)

const (
	pluginType = "vpa.v2beta2.autoscaling.k8s.io"
)

type partition string
type pod_name string

var _ skplug.Plugin = &pluginServer{}

type pluginServer struct {
	mux         sync.RWMutex
	autoscalers map[partition]*vpaplugin.Autoscaler
}

func newPluginServer() *pluginServer {
	return &pluginServer{
		autoscalers: make(map[partition]*vpaplugin.Autoscaler),
	}
}

func (p *pluginServer) Event(part string, time int64, typ proto.EventType, object skplug.Object) error {
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

func (p *pluginServer) Stat(part string, stat []*proto.Stat) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	a, ok := p.autoscalers[partition(part)]
	if !ok {
		return fmt.Errorf("stat for non-existant autoscaler partition: %v", part)
	}
	return a.Stat(stat)
}

func (p *pluginServer) HorizontalRecommendation(part string, time int64) (rec int32, err error) {
	panic("unimplemented")
}

func (p *pluginServer) VerticalRecommendation(part string, time int64) (rec []*proto.RecommendedPodResources, err error) {
	p.mux.Lock()
	defer p.mux.Unlock()
	a, ok := p.autoscalers[partition(part)]
	if !ok {
		return []*proto.RecommendedPodResources{}, fmt.Errorf("scale for non-existant autoscaler partition: %v", part)
	}
	return a.VerticalRecommendation(time)
}

func (p *pluginServer) createAutoscaler(part partition, a *skplug.Autoscaler) error {
	if a.Type != pluginType {
		return fmt.Errorf("unsupported autoscaler type %v. this plugin supports %v", a.Type, pluginType)
	}

	p.mux.Lock()
	defer p.mux.Unlock()
	if _, ok := p.autoscalers[part]; ok {
		return fmt.Errorf("duplicate create autoscaler event in partition %v", part)
	}
	autoscaler, err := vpaplugin.NewAutoscaler(a.Yaml)
	if err != nil {
		return err
	}
	p.autoscalers[part] = autoscaler
	log.Println("created autoscaler", part)
	return nil
}

func (p *pluginServer) deleteAutoscaler(part partition) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	autoscaler, ok := p.autoscalers[part]
	if !ok {
		return fmt.Errorf("delete autoscaler event for non-existant partition %v", part)
	}
	log.Printf("final vpa state: %v", autoscaler.String())
	delete(p.autoscalers, part)
	log.Println("deleted autoscaler", part)
	return nil
}

func (p *pluginServer) createPod(part partition, pod *skplug.Pod) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	autoscaler, ok := p.autoscalers[part]
	if !ok {
		return fmt.Errorf("create pod event for non-existant partition %v", part)
	}
	return autoscaler.CreatePod((*proto.Pod)(pod))
}

func (p *pluginServer) updatePod(part partition, pod *skplug.Pod) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	autoscaler, ok := p.autoscalers[part]
	if !ok {
		return fmt.Errorf("update pod event for non-existant partition %v", part)
	}
	return autoscaler.UpdatePod((*proto.Pod)(pod))
}

func (p *pluginServer) deletePod(part partition, pod *skplug.Pod) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	autoscaler, ok := p.autoscalers[part]
	if !ok {
		return fmt.Errorf("delete pod event for non-existant partition %v", part)
	}
	return autoscaler.DeletePod((*proto.Pod)(pod))
}

func main() {
	klog.InitFlags(flag.CommandLine)
	klog.Infof("Starting Skenario Kubernetes VPA plugin.")
	//test := newPluginServer()
	//test.createAutoscaler("1", &skplug.Autoscaler{
	//	// TODO: select type and plugin based on the scenario.
	//	Type: "vpa.v2beta2.autoscaling.k8s.io",
	//	Yaml: vpaYaml,
	//})
	//test.VerticalRecommendation("1", time.Now().UnixNano())
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: skplug.Handshake,
		Plugins: map[string]plugin.Plugin{
			"autoscaler": &skplug.AutoscalerPlugin{Impl: newPluginServer()},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

//const vpaYaml = `
//apiVersion: autoscaling.k8s.io/v1
//kind: VerticalPodAutoscaler
//metadata:
//name: my-rec-vpa
//spec:
//targetRef:
//  apiVersion: "apps/v1"
//  kind:       Deployment
//  name:       my-rec-deployment
//updatePolicy:
//  updateMode: "Off"
//`
