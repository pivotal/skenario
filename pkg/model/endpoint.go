/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package model

import (
	"net"
	"time"

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

const (
	StateEndpointNonexistent = "EndpointNonexistent"
	StateEndpointActive = "EndpointActive"

	addEndpoint    = "add_endpoint"
	removeEndpoint = "remove_endpoint"
)

func (re *ReplicaEndpoints) Identity() simulator.ProcessIdentity {
	return re.name
}

func (re *ReplicaEndpoints) OnOccurrence(event simulator.Event) (result simulator.StateTransitionResult) {
	var from, to string

	switch event.Name() {
	case addEndpoint:
		re.nextAddress[3]++ // TODO: what happens when this overflows?
		newAddress := corev1.EndpointAddress{
			IP:       re.nextAddress.String(),
			Hostname: string(event.SubjectIdentity()),
		}
		newSubset := corev1.EndpointSubset{
			Addresses: []corev1.EndpointAddress{newAddress},
		}
		newEndpoints := &corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{newSubset},
		}

		re.replicaEndpoints[event.SubjectIdentity()] = newEndpoints

		re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Create(newEndpoints)
		re.endpointsInformer.Informer().GetIndexer().Add(newEndpoints)

		from = StateEndpointNonexistent
		to = StateEndpointActive

	case removeEndpoint:
		re.nextAddress[3]--
		endpoint := re.replicaEndpoints[event.SubjectIdentity()]
		delete(re.replicaEndpoints, event.SubjectIdentity())

		grace := int64(0)
		propPolicy := metav1.DeletePropagationForeground
		err := re.kubernetesClient.CoreV1().Endpoints(simulatorNamespace).Delete(endpoint.Name, &v1.DeleteOptions{
			GracePeriodSeconds: &grace,
			PropagationPolicy:  &propPolicy,
		})
		if err != nil {
			panic(err.Error())
		}

		err = re.endpointsInformer.Informer().GetIndexer().Delete(endpoint)
		if err != nil {
			panic(err.Error())
		}

		from = StateEndpointActive
		to = StateEndpointNonexistent
	}

	return simulator.StateTransitionResult{FromState: from, ToState: to}
}

func (re *ReplicaEndpoints) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunchingReplica:
		re.env.Schedule(simulator.NewGeneralEvent(
			addEndpoint,
			event.OccursAt().Add(1*time.Nanosecond),
			re,
		))
	case terminateReplica:
		re.env.Schedule(simulator.NewGeneralEvent(
			addEndpoint,
			event.OccursAt().Add(-1*time.Nanosecond),
			re,
		))
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
