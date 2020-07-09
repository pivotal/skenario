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

	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8sfakes "k8s.io/client-go/kubernetes/fake"

	"skenario/pkg/simulator"
)

type ClusterConfig struct {
	LaunchDelay             time.Duration
	TerminateDelay          time.Duration
	NumberOfRequests        uint
	InitialNumberOfReplicas uint
}

type ClusterModel interface {
	Model
	Desired() ReplicasDesiredStock
	CurrentLaunching() uint64
	CurrentActive() uint64
	RecordToAutoscaler(atTime *time.Time)
	RoutingStock() RequestsRoutingStock
	ActiveStock() simulator.ThroughStock
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
	requestsInRouting   simulator.ThroughStock
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

func (cm *clusterModel) RecordToAutoscaler(atTime *time.Time) {
	// first report for the RoutingStock
	stats := make([]*proto.Stat, 0)
	stats = append(stats, &proto.Stat{
		Time:    atTime.UnixNano(),
		PodName: "RoutingStock",
		Type:    proto.MetricType_CONCURRENT_REQUESTS_MILLIS,
		Value:   int32(cm.requestsInRouting.Count() * 1000),
	})
	// TODO: report request count

	// and then report for the replicas
	for _, e := range cm.replicasActive.EntitiesInStock() {
		r := (*e).(ReplicaEntity)
		stats = append(stats, r.Stats()...)
	}
	err := cm.env.Plugin().Stat(stats)
	if err != nil {
		panic(err)
	}
}

func (cm *clusterModel) EPInformer() corev1informers.EndpointsInformer {
	return cm.endpointsInformer
}

func (cm *clusterModel) RoutingStock() RequestsRoutingStock {
	return cm.requestsInRouting
}

func (cm *clusterModel) ActiveStock() simulator.ThroughStock {
	return cm.replicasActive
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

	replicasActive := NewReplicasActiveStock(env)
	requestsFailed := simulator.NewSinkStock("RequestsFailed", "Request")
	routingStock := NewRequestsRoutingStock(env, replicasActive, requestsFailed)
	replicasTerminated := simulator.NewSinkStock("ReplicasTerminated", simulator.EntityKind("Replica"))

	cm := &clusterModel{
		env:                 env,
		config:              config,
		replicasConfig:      replicasConfig,
		replicaSource:       NewReplicaSource(env, fakeClient, endpointsInformer, replicasConfig.MaxRPS),
		replicasLaunching:   simulator.NewThroughStock("ReplicasLaunching", simulator.EntityKind("Replica")),
		replicasActive:      replicasActive,
		replicasTerminating: NewReplicasTerminatingStock(env, replicasConfig, replicasTerminated),
		replicasTerminated:  replicasTerminated,
		requestsInRouting:   routingStock,
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
