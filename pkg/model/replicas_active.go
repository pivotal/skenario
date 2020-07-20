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
	"skenario/pkg/simulator"
)

type ReplicasActiveStock interface {
	simulator.ThroughStock
}

type replicasActiveStock struct {
	env      simulator.Environment
	delegate simulator.ThroughStock
	replicas []*ReplicaEntity
}

func (ras *replicasActiveStock) Name() simulator.StockName {
	return ras.delegate.Name()
}

func (ras *replicasActiveStock) KindStocked() simulator.EntityKind {
	return ras.delegate.KindStocked()
}

func (ras *replicasActiveStock) Count() uint64 {
	return ras.delegate.Count()
}

func (ras *replicasActiveStock) EntitiesInStock() map[simulator.Entity]bool {
	return ras.delegate.EntitiesInStock()
}

func (ras *replicasActiveStock) GetEntityByNumber(number int) simulator.Entity {
	return *ras.replicas[number]
}

func (ras *replicasActiveStock) Remove(entity *simulator.Entity) simulator.Entity {
	removedEntity := ras.delegate.Remove(entity)
	if removedEntity == nil {
		return nil
	}

	replica := removedEntity.(ReplicaEntity)
	replica.Deactivate()

	//support replicas array updated
	ras.deleteReplica(&replica)
	return removedEntity
}

func (ras *replicasActiveStock) Add(entity simulator.Entity) error {
	replica := entity.(ReplicaEntity)
	replica.Activate()
	ras.replicas = append(ras.replicas, &replica)
	return ras.delegate.Add(entity)
}

func (ras *replicasActiveStock) deleteReplica(replica *ReplicaEntity) {
	removed := false
	for i := 0; i < len(ras.replicas); i++ {
		if ras.replicas[i] == replica {
			ras.replicas[i] = ras.replicas[len(ras.replicas)-1]
			removed = true
		}
	}
	if removed {
		ras.replicas = ras.replicas[:len(ras.replicas)-1]
	}
}

func NewReplicasActiveStock(env simulator.Environment) ReplicasActiveStock {
	return &replicasActiveStock{
		env:      env,
		delegate: simulator.NewThroughStock("ReplicasActive", "Replica"),
		replicas: make([]*ReplicaEntity, 0),
	}
}
