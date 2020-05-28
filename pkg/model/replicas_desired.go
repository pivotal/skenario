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

	"skenario/pkg/simulator"
)

type ReplicasConfig struct {
	LaunchDelay    time.Duration
	TerminateDelay time.Duration
	MaxRPS         int64
}

type RequestConfig struct {
	CPUUtilization int
	IOUtilization  int
	Timeout        time.Duration
}

type ReplicasDesiredStock interface {
	simulator.ThroughStock
}

type replicasDesiredStock struct {
	env                 simulator.Environment
	config              ReplicasConfig
	delegate            simulator.ThroughStock
	replicaSource       ReplicaSource
	replicasLaunching   simulator.ThroughStock
	replicasActive      simulator.ThroughStock
	replicasTerminating ReplicasTerminatingStock
	launchingCount      uint64
}

func (rds *replicasDesiredStock) Name() simulator.StockName {
	return rds.delegate.Name()
}

func (rds *replicasDesiredStock) KindStocked() simulator.EntityKind {
	return rds.delegate.KindStocked()
}

func (rds *replicasDesiredStock) Count() uint64 {
	return rds.delegate.Count()
}

func (rds *replicasDesiredStock) EntitiesInStock() []*simulator.Entity {
	return rds.delegate.EntitiesInStock()
}

func (rds *replicasDesiredStock) Remove() simulator.Entity {
	ent := rds.delegate.Remove()
	if ent == nil {
		return nil
	}

	nextTerminate := rds.env.CurrentMovementTime().Add(1 * time.Nanosecond)
	if rds.replicasLaunching.Count() > 0 {
		rds.env.AddToSchedule(simulator.NewMovement(
			"terminate_launch",
			nextTerminate,
			rds.replicasLaunching,
			rds.replicasTerminating,
		))
	} else {
		rds.env.AddToSchedule(simulator.NewMovement(
			"terminate_active",
			nextTerminate,
			rds.replicasActive,
			rds.replicasTerminating,
		))
	}

	return ent
}

func (rds *replicasDesiredStock) Add(entity simulator.Entity) error {
	err := rds.delegate.Add(entity)
	if err != nil {
		return err
	}

	rds.env.AddToSchedule(simulator.NewMovement(
		"begin_launch",
		rds.env.CurrentMovementTime().Add(1*time.Nanosecond),
		rds.replicaSource,
		rds.replicasLaunching,
	))

	rds.env.AddToSchedule(simulator.NewMovement(
		"finish_launching",
		rds.env.CurrentMovementTime().Add(rds.config.LaunchDelay),
		rds.replicasLaunching,
		rds.replicasActive,
	))

	return nil
}

func NewReplicasDesiredStock(env simulator.Environment, config ReplicasConfig, replicaSource ReplicaSource, replicasLaunching, replicasActive simulator.ThroughStock, replicasTerminating ReplicasTerminatingStock) ReplicasDesiredStock {
	return &replicasDesiredStock{
		env:                 env,
		config:              config,
		delegate:            simulator.NewThroughStock("ReplicasDesired", "Desired"),
		replicaSource:       replicaSource,
		replicasLaunching:   replicasLaunching,
		replicasActive:      replicasActive,
		replicasTerminating: replicasTerminating,
	}
}
