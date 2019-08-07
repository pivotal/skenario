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
	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/autoscaler"

	"time"

	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	"skenario/pkg/simulator"
)

type ClusterConfig struct {
	LaunchDelay      time.Duration
	TerminateDelay   time.Duration
	NumberOfRequests uint

	KnativeAutoscalerSpecific
}

type ClusterModel interface {
	Model
	Desired() ReplicasDesiredStock
	CurrentLaunching() uint64
	CurrentActive() uint64
	BufferStock() RequestsBufferedStock
	ActiveStock() ReplicasActiveStock
	Collector() *autoscaler.MetricCollector
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
	collector           *autoscaler.MetricCollector
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

func (cm *clusterModel) BufferStock() RequestsBufferedStock {
	return cm.requestsInBuffer
}

func (cm *clusterModel) ActiveStock() ReplicasActiveStock {
	return cm.replicasActive
}

func (cm *clusterModel) Collector() *autoscaler.MetricCollector {
	return cm.collector
}

func NewCluster(env simulator.Environment, config ClusterConfig, replicasConfig ReplicasConfig) ClusterModel {
	replicasActive := NewReplicasActiveStock()
	requestsFailed := simulator.NewSinkStock("RequestsFailed", "Request")
	replicasTerminated := simulator.NewSinkStock("ReplicasTerminated", simulator.EntityKind("Replica"))

	logger := logging.FromContext(env.Context())
	collector := NewMetricCollector(logger, KnativeAutoscalerConfig{
		DeciderSpec: autoscaler.DeciderSpec{
			StableWindow: 60 * time.Second,
		},
		KnativeAutoscalerSpecific: config.KnativeAutoscalerSpecific,
	}, replicasActive)
	bufferStock := NewRequestsBufferedStock(env, replicasActive, requestsFailed, collector)

	cm := &clusterModel{
		env:                 env,
		config:              config,
		replicasConfig:      replicasConfig,
		replicaSource:       NewReplicaSource(env, replicasConfig.MaxRPS),
		replicasLaunching:   simulator.NewThroughStock("ReplicasLaunching", simulator.EntityKind("Replica")),
		replicasActive:      replicasActive,
		replicasTerminating: NewReplicasTerminatingStock(env, replicasConfig, replicasTerminated),
		replicasTerminated:  replicasTerminated,
		requestsInBuffer:    bufferStock,
		requestsFailed:      requestsFailed,
		collector:           collector,
	}

	desiredConf := ReplicasConfig{
		LaunchDelay:    config.LaunchDelay,
		TerminateDelay: config.TerminateDelay,
	}

	cm.replicasDesired = NewReplicasDesiredStock(env, desiredConf, cm.replicaSource, cm.replicasLaunching, cm.replicasActive, cm.replicasTerminating)

	return cm
}
