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
	"time"
)

type ReplicasActiveStock interface {
	simulator.ThroughStock
}

type replicasActiveStock struct {
	env      simulator.Environment
	delegate simulator.ThroughStock
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

func (ras *replicasActiveStock) EntitiesInStock() []*simulator.Entity {
	return ras.delegate.EntitiesInStock()
}

func (ras *replicasActiveStock) Remove(entity *simulator.Entity) simulator.Entity {
	removedEntity := ras.delegate.Remove(entity)
	if removedEntity == nil {
		return nil
	}

	replica := removedEntity.(ReplicaEntity)
	replica.Deactivate()
	return removedEntity
}

func (ras *replicasActiveStock) Add(entity simulator.Entity) error {
	replica := entity.(Replica)
	replica.Activate()
	ras.env.AddToSchedule(simulator.NewMovement(
		"metrics_tick",
		ras.env.CurrentMovementTime().Add(5*time.Second),
		replica.MetricsTicktock(),
		replica.MetricsTicktock(),
		&entity))
	return ras.delegate.Add(entity)
}

func NewReplicasActiveStock(env simulator.Environment) ReplicasActiveStock {
	return &replicasActiveStock{
		env:      env,
		delegate: simulator.NewArrayThroughStock("ReplicasActive", "Replica"),
	}
}
