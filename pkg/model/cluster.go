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
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8sfakes "k8s.io/client-go/kubernetes/fake"

	"skenario/pkg/simulator"
)

type ClusterConfig struct {
	LaunchDelay      time.Duration
	TerminateDelay   time.Duration
	NumberOfRequests uint
}

type ClusterModel interface {
	Model
	Desired() ReplicasDesiredStock
	CurrentLaunching() uint64
	CurrentActive() uint64
	RecordToAutoscaler(scaler autoscaler.UniScaler, atTime *time.Time)
	BufferStock() RequestsBufferedStock
}

type EndpointInformerSource interface {
	EPInformer() corev1informers.EndpointsInformer
}

type clusterModel struct {
	env                 simulator.Environment
	config              ClusterConfig
	replicasConfig      ReplicasConfig
	replicasDesired     ReplicasDesiredStock
	replicaSource       ReplicaSource
	replicasLaunching   simulator.ThroughStock
	replicasActive      simulator.ThroughStock
	replicasTerminating ReplicasTerminatingStock
	replicasTerminated  simulator.SinkStock
	requestsInBuffer    simulator.ThroughStock
	requestsFailed      simulator.SinkStock
	kubernetesClient    kubernetes.Interface
	endpointsInformer   corev1informers.EndpointsInformer
}

func (cm *clusterModel) Env() simulator.Environment {
	return cm.env
}

func (cm *clusterModel) Desired() ReplicasDesiredStock {
	return cm.replicasDesired
}
func (cm *clusterModel) CurrentLaunching() uint64 {
	return cm.replicasLaunching.Count()
}

func (cm *clusterModel) CurrentActive() uint64 {
	return cm.replicasActive.Count()
}

func (cm *clusterModel) RecordToAutoscaler(scaler autoscaler.UniScaler, atTime *time.Time) {
	// first report for the buffer
	scaler.Record(cm.env.Context(), autoscaler.Stat{
		Time:                      atTime,
		PodName:                   "Buffer",
		AverageConcurrentRequests: float64(cm.requestsInBuffer.Count()),
		RequestCount:              int32(cm.requestsInBuffer.Count()),
	})

	// and then report for the replicas
	for _, e := range cm.replicasActive.EntitiesInStock() {
		r := (*e).(ReplicaEntity)
		stat := r.Stat()

		scaler.Record(cm.env.Context(), stat)
	}
}

func (cm *clusterModel) EPInformer() corev1informers.EndpointsInformer {
	return cm.endpointsInformer
}

func (cm *clusterModel) BufferStock() RequestsBufferedStock {
	return cm.requestsInBuffer
}

func NewCluster(env simulator.Environment, config ClusterConfig, replicasConfig ReplicasConfig) ClusterModel {
	fakeClient := k8sfakes.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)
	endpointsInformer := informerFactory.Core().V1().Endpoints()

	newEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "Skenario Revision",
		},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{},
		}},
	}

	fakeClient.CoreV1().Endpoints("skenario").Create(newEndpoints)
	endpointsInformer.Informer().GetIndexer().Add(newEndpoints)

	replicasActive := NewReplicasActiveStock()
	requestsFailed := simulator.NewSinkStock("RequestsFailed", "Request")
	bufferStock := NewRequestsBufferedStock(env, replicasActive, requestsFailed)
	replicasTerminated := simulator.NewSinkStock("ReplicasTerminated", simulator.EntityKind("Replica"))

	cm := &clusterModel{
		env:                 env,
		config:              config,
		replicasConfig:		 replicasConfig,
		replicaSource:       NewReplicaSource(env, fakeClient, endpointsInformer, replicasConfig.MaxRPS),
		replicasLaunching:   simulator.NewThroughStock("ReplicasLaunching", simulator.EntityKind("Replica")),
		replicasActive:      replicasActive,
		replicasTerminating: NewReplicasTerminatingStock(env, replicasConfig, replicasTerminated),
		replicasTerminated:  replicasTerminated,
		requestsInBuffer:    bufferStock,
		requestsFailed:      requestsFailed,
		kubernetesClient:    fakeClient,
		endpointsInformer:   endpointsInformer,
	}

	desiredConf := ReplicasConfig{
		LaunchDelay:    config.LaunchDelay,
		TerminateDelay: config.TerminateDelay,
	}

	cm.replicasDesired = NewReplicasDesiredStock(env, desiredConf, cm.replicaSource, cm.replicasLaunching, cm.replicasActive, cm.replicasTerminating)

	return cm
}
