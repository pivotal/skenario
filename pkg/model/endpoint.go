package model

import (
	"net"
	"time"

	"github.com/looplab/fsm"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"knative-simulator/pkg/simulator"

	corev1 "k8s.io/api/core/v1"
	informersCoreV1 "k8s.io/client-go/informers/core/v1"
)

type ReplicaEndpoints struct {
	name simulator.ProcessIdentity
	env  *simulator.Environment
	fsm  *fsm.FSM

	endpoints *corev1.Endpoints

	kubernetesClient  kubernetes.Interface
	informerFactory   *informers.SharedInformerFactory
	endpointsInformer informersCoreV1.EndpointsInformer

	nextAddress net.IP
}

const (
	StateEndpointsActive          = "EndpointsActive"
	StateEndpointsAddingAddress   = "EndpointsAddingAddress"
	StateEndpointsRemovingAddress = "EndpointsRemovingAddress"

	addEndpointAddress            = "add_address"
	finishAddingEndpointAddress   = "finish_adding_address"
	removeEndpointAddress         = "remove_address"
	finishRemovingEndpointAddress = "finish_removing_address"
)

func (re *ReplicaEndpoints) Identity() simulator.ProcessIdentity {
	return re.name
}

func (re *ReplicaEndpoints) OnOccurrence(event *simulator.Event) (result simulator.TransitionResult) {
	switch event.Name {
	case addEndpointAddress:
		re.nextAddress[3]++ // TODO: what happens when this overflows?
		newAddress := corev1.EndpointAddress{
			IP:       re.nextAddress.String(),
			Hostname: string(event.Subject.Identity()),
		}
		newSubset := corev1.EndpointSubset{
			Addresses: []corev1.EndpointAddress{newAddress},
		}
		re.endpoints.Subsets = append(re.endpoints.Subsets, newSubset)

		re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Update(re.endpoints)
		re.endpointsInformer.Informer().GetIndexer().Update(re.endpoints)

		re.env.Schedule(&simulator.Event{
			Name:     finishAddingEndpointAddress,
			OccursAt: event.OccursAt.Add(1 * time.Millisecond),
			Subject:  re,
		})
	case finishAddingEndpointAddress:
		// nothing
	case removeEndpointAddress:
		re.nextAddress[3]--
		re.endpoints.Subsets = re.endpoints.Subsets[:len(re.endpoints.Subsets)-1]

		re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Update(re.endpoints)
		re.endpointsInformer.Informer().GetIndexer().Update(re.endpoints)
		re.env.Schedule(&simulator.Event{
			Name:     finishRemovingEndpointAddress,
			OccursAt: event.OccursAt.Add(1 * time.Millisecond),
			Subject:  re,
		})
	case finishRemovingEndpointAddress:
		// nothing
	}

	currentState := re.fsm.Current()
	err := re.fsm.Event(string(event.Name))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			panic(err.Error())
		}
	}

	return simulator.TransitionResult{FromState: currentState, ToState: re.fsm.Current()}
}

func (re *ReplicaEndpoints) OnSchedule(event *simulator.Event) {
	switch event.Name {
	case finishLaunchingReplica:
		re.env.Schedule(&simulator.Event{
			Name:     addEndpointAddress,
			OccursAt: event.OccursAt.Add(-50 * time.Millisecond),
			Subject:  re,
		})
	case terminateReplica:
		re.env.Schedule(&simulator.Event{
			Name:     removeEndpointAddress,
			OccursAt: event.OccursAt.Add(50 * time.Millisecond),
			Subject:  re,
		})
	}
}

func (re *ReplicaEndpoints) AddRevisionReplica(replica *RevisionReplica) {
	re.env.ListenForScheduling(replica.Identity(), finishLaunchingReplica, re)
	re.env.ListenForScheduling(replica.Identity(), terminateReplica, re)
}

func NewReplicaEndpoints(name simulator.ProcessIdentity, env *simulator.Environment, kubernetesClient kubernetes.Interface) *ReplicaEndpoints {
	informerFactory := informers.NewSharedInformerFactory(kubernetesClient, 0)
	endpointsInformer := informerFactory.Core().V1().Endpoints()

	re := &ReplicaEndpoints{
		name:              name,
		env:               env,
		endpoints:         &corev1.Endpoints{},
		kubernetesClient:  kubernetesClient,
		informerFactory:   &informerFactory,
		endpointsInformer: endpointsInformer,
		nextAddress:       net.IPv4(192, 168, 0, 0),
	}

	re.fsm = fsm.NewFSM(
		StateEndpointsActive,
		fsm.Events{ // TODO: do I care about the final cleanup of Endpoints?
			//fsm.EventDesc{Name: createEndpoints, Src: []string{StateEndpointsNotCreated}, Dst: StateEndpointsCreating},
			//fsm.EventDesc{Name: finishCreatingEndpoints, Src: []string{StateEndpointsCreating}, Dst: StateEndpointsActive},
			fsm.EventDesc{Name: addEndpointAddress, Src: []string{StateEndpointsActive}, Dst: StateEndpointsAddingAddress},
			fsm.EventDesc{Name: finishAddingEndpointAddress, Src: []string{StateEndpointsAddingAddress}, Dst: StateEndpointsActive},
			fsm.EventDesc{Name: removeEndpointAddress, Src: []string{StateEndpointsActive}, Dst: StateEndpointsRemovingAddress},
			fsm.EventDesc{Name: finishRemovingEndpointAddress, Src: []string{StateEndpointsRemovingAddress}, Dst: StateEndpointsActive},
		},
		fsm.Callbacks{},
	)

	// Create empty endpoints
	re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Update(re.endpoints)
	endpointsInformer.Informer().GetIndexer().Add(re.endpoints)

	return re
}
