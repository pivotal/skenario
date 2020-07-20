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
	"skenario/pkg/simulator"
	"time"
)

type ReplicasTerminatingStock interface {
	simulator.ThroughStock
}

type replicasTerminatingStock struct {
	env                simulator.Environment
	config             ReplicasConfig
	delegate           simulator.ThroughStock
	replicasTerminated simulator.SinkStock
}

func (rts *replicasTerminatingStock) Name() simulator.StockName {
	return rts.delegate.Name()
}

func (rts *replicasTerminatingStock) KindStocked() simulator.EntityKind {
	return rts.delegate.KindStocked()
}

func (rts *replicasTerminatingStock) Count() uint64 {
	return rts.delegate.Count()
}

func (rts *replicasTerminatingStock) EntitiesInStock() map[simulator.Entity]bool {
	return rts.delegate.EntitiesInStock()
}
func (rts *replicasTerminatingStock) GetEntityByNumber(number int) simulator.Entity {
	return rts.delegate.GetEntityByNumber(number)
}
func (rts *replicasTerminatingStock) Remove(entity *simulator.Entity) simulator.Entity {
	return rts.delegate.Remove(entity)
}

func (rts *replicasTerminatingStock) Add(entity simulator.Entity) error {
	err := rts.delegate.Add(entity)
	if err != nil {
		return fmt.Errorf("could not add entity (%+v) to ReplicasTerminating stock: %s", entity, err.Error())
	}

	replica := entity.(Replica)
	count := replica.RequestsProcessing().Count()
	drainTime := time.Second * time.Duration(count)

	terminateAt := rts.env.CurrentMovementTime().Add(drainTime).Add(rts.config.TerminateDelay)
	rts.env.AddToSchedule(simulator.NewMovement(
		"finish_terminating",
		terminateAt,
		rts.delegate,
		rts.replicasTerminated,
		&entity,
	))

	return nil
}

func NewReplicasTerminatingStock(env simulator.Environment, config ReplicasConfig, replicasTerminated simulator.SinkStock) ReplicasTerminatingStock {
	return &replicasTerminatingStock{
		env:                env,
		config:             config,
		delegate:           simulator.NewThroughStock("ReplicasTerminating", "Replica"),
		replicasTerminated: replicasTerminated,
	}
}
