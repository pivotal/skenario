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
	"fmt"

	"github.com/knative/serving/pkg/autoscaler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	"skenario/pkg/simulator"
)

type Replica interface {
	Activate()
	Deactivate()
	RequestsProcessing() RequestsProcessingStock
	Stat() autoscaler.Stat
}

type ReplicaEntity interface {
	simulator.Entity
	Replica
}

type replicaEntity struct {
	env                  simulator.Environment
	number               int
	kubernetesClient     kubernetes.Interface
	endpointsInformer    informers.EndpointsInformer
	endpointAddress      corev1.EndpointAddress
	requestsProcessing   RequestsProcessingStock
	requestsComplete     simulator.SinkStock
	numRequestsSinceStat int32
}

var replicaNum int

func (re *replicaEntity) Activate() {
	endpoints, err := re.kubernetesClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	endpoints.Subsets[0].Addresses = append(endpoints.Subsets[0].Addresses, re.endpointAddress)

	updatedEndpoints, err := re.kubernetesClient.CoreV1().Endpoints("skenario").Update(endpoints)
	if err != nil {
		panic(err.Error())
	}
	err = re.endpointsInformer.Informer().GetIndexer().Update(updatedEndpoints)
	if err != nil {
		panic(err.Error())
	}
}

func (re *replicaEntity) Deactivate() {
	endpoints, err := re.kubernetesClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	addresses := endpoints.Subsets[0].Addresses

	for i, addr := range addresses {
		if addr == re.endpointAddress {
			// remove by swapping with last entry and then truncating
			addresses[i] = addresses[len(addresses)-1]
			addresses = addresses[:len(addresses)-1]

			break
		}
	}

	endpoints.Subsets[0].Addresses = addresses

	updatedEndpoints, err := re.kubernetesClient.CoreV1().Endpoints("skenario").Update(endpoints)
	if err != nil {
		panic(err.Error())
	}
	err = re.endpointsInformer.Informer().GetIndexer().Update(updatedEndpoints)
	if err != nil {
		panic(err.Error())
	}
}

func (re *replicaEntity) RequestsProcessing() RequestsProcessingStock {
	return re.requestsProcessing
}

func (re *replicaEntity) Stat() autoscaler.Stat {
	atTime := re.env.CurrentMovementTime()
	stat := autoscaler.Stat{
		Time:                      &atTime,
		PodName:                   string(re.Name()),
		AverageConcurrentRequests: float64(re.requestsProcessing.Count()),
		RequestCount:              re.requestsProcessing.RequestCount(),
	}

	re.numRequestsSinceStat = 0

	return stat
}

func (re *replicaEntity) Name() simulator.EntityName {
	return simulator.EntityName(fmt.Sprintf("replica-%d", re.number))
}

func (re *replicaEntity) Kind() simulator.EntityKind {
	return "Replica"
}

func NewReplicaEntity(env simulator.Environment, client kubernetes.Interface, endpointsInformer informers.EndpointsInformer, address string) ReplicaEntity {
	replicaNum++
	complete := simulator.NewSinkStock("RequestsComplete", "Request")

	re := &replicaEntity{
		env:               env,
		number:            replicaNum,
		kubernetesClient:  client,
		endpointsInformer: endpointsInformer,
		requestsComplete:  complete,
	}

	re.requestsProcessing = NewRequestsProcessingStock(env, re.Name(), complete)

	re.endpointAddress = corev1.EndpointAddress{
		IP:       address,
		Hostname: string(re.Name()),
	}

	return re
}
