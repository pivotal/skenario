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
	"encoding/binary"
	"net"

	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	"skenario/pkg/simulator"
)

type IPV4Sequence interface {
	Next() string
}

type ReplicaSource interface {
	simulator.SourceStock
}

type replicaSource struct {
	env               simulator.Environment
	kubernetesClient  kubernetes.Interface
	endpointsInformer corev1informers.EndpointsInformer
	nextIPValue       uint32
	maxReplicaRPS     int64
}

func (rs *replicaSource) Name() simulator.StockName {
	return "ReplicaSource"
}

func (rs *replicaSource) KindStocked() simulator.EntityKind {
	return "Replica"
}

func (rs *replicaSource) Count() uint64 {
	return 0
}

func (rs *replicaSource) EntitiesInStock() []*simulator.Entity {
	return []*simulator.Entity{}
}

func (rs *replicaSource) Remove() simulator.Entity {
	return NewReplicaEntity(rs.env, rs.kubernetesClient, rs.endpointsInformer, rs.Next(), rs.maxReplicaRPS)
}

func (rs *replicaSource) Next() string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, rs.nextIPValue)

	rs.nextIPValue++

	return ip.String()
}

func NewReplicaSource(env simulator.Environment, maxReplicaRPS int64) ReplicaSource {
	return &replicaSource{
		env:               env,
		nextIPValue:       1,
		maxReplicaRPS: maxReplicaRPS,
	}
}
