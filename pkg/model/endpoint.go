package model

import (
	"net"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"knative-simulator/pkg/simulator"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informersCoreV1 "k8s.io/client-go/informers/core/v1"
)

type ReplicaEndpoints struct {
	name simulator.ProcessIdentity
	env  *simulator.Environment

	replicaEndpoints map[simulator.ProcessIdentity]*corev1.Endpoints

	kubernetesClient  kubernetes.Interface
	informerFactory   *informers.SharedInformerFactory
	endpointsInformer informersCoreV1.EndpointsInformer

	nextAddress net.IP
}

func (re *ReplicaEndpoints) Identity() simulator.ProcessIdentity {
	return re.name
}

func (re *ReplicaEndpoints) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunchingReplica:
		re.nextAddress[3]++ // TODO: what happens when this overflows?
		newAddress := corev1.EndpointAddress{
			IP:       re.nextAddress.String(),
			Hostname: string(event.Subject().Identity()),
		}
		newSubset := corev1.EndpointSubset{
			Addresses: []corev1.EndpointAddress{newAddress},
		}
		newEndpoints := &corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{newSubset},
		}

		re.replicaEndpoints[event.Subject().Identity()] = newEndpoints

		re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Create(newEndpoints)
		re.endpointsInformer.Informer().GetIndexer().Add(newEndpoints)
	case terminateReplica:
		re.nextAddress[3]--
		endpoint := re.replicaEndpoints[event.Subject().Identity()]
		delete(re.replicaEndpoints, event.Subject().Identity())

		grace := int64(0)
		propPolicy := metav1.DeletePropagationForeground
		re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Delete(endpoint.Name, &v1.DeleteOptions{
			GracePeriodSeconds: &grace,
			PropagationPolicy:  &propPolicy,
		})
		re.endpointsInformer.Informer().GetIndexer().Delete(endpoint)
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
		replicaEndpoints:  make(map[simulator.ProcessIdentity]*corev1.Endpoints),
		kubernetesClient:  kubernetesClient,
		informerFactory:   &informerFactory,
		endpointsInformer: endpointsInformer,
		nextAddress:       net.IPv4(192, 168, 0, 0),
	}

	return re
}
