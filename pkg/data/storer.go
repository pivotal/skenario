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

package data

import (
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type Storer interface {
	Store(
		dbFileName string,
		completed []simulator.CompletedMovement,
		ignored []simulator.IgnoredMovement,
		clusterConf model.ClusterConfig,
		kpaConf model.KnativeAutoscalerConfig,
	) error
}

type storer struct {
}

func (s *storer) Store(
	dbFileName string,
	completed []simulator.CompletedMovement,
	ignored []simulator.IgnoredMovement,
	clusterConf model.ClusterConfig,
	kpaConf model.KnativeAutoscalerConfig,
) error {
	conn, err := sqlite3.Open(dbFileName)
	if err != nil {
		return err
	}

	err = conn.Exec(Schema)
	if err != nil {
		return err
	}
	defer conn.Close()

	srStmt, err := conn.Prepare(`insert into scenario_runs( 
									   recorded
									 , origin
									 , traffic_pattern
									 , cluster_launch_delay
									 , cluster_terminate_delay
									 , cluster_number_of_requests
									 , autoscaler_tick_interval
									 , autoscaler_stable_window
									 , autoscaler_panic_window
									 , autoscaler_scale_to_zero_grace_period
									 , autoscaler_target_concurrency_default
									 , autoscaler_target_concurrency_percentage
									 , autoscaler_max_scale_up_rate)
									values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return err
	}

	err = srStmt.Exec(
		time.Now().Format(time.RFC3339),
		"skenario_cli",
		"golang_rand_uniform",
		clusterConf.LaunchDelay.Nanoseconds(),
		clusterConf.TerminateDelay.Nanoseconds(),
		int(clusterConf.NumberOfRequests),
		kpaConf.TickInterval.Nanoseconds(),
		kpaConf.StableWindow.Nanoseconds(),
		kpaConf.PanicWindow.Nanoseconds(),
		kpaConf.ScaleToZeroGracePeriod.Nanoseconds(),
		kpaConf.TargetConcurrencyDefault,
		kpaConf.TargetConcurrencyPercentage,
		kpaConf.MaxScaleUpRate,
	)
	if err != nil {
		return err
	}

	return nil
}

