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
	conn        *sqlite3.Conn
	clusterConf model.ClusterConfig
	kpaConf     model.KnativeAutoscalerConfig
	completed   []simulator.CompletedMovement
	ignored     []simulator.IgnoredMovement
}

func (s *storer) Store(
	dbFileName string,
	completed []simulator.CompletedMovement,
	ignored []simulator.IgnoredMovement,
	clusterConf model.ClusterConfig,
	kpaConf model.KnativeAutoscalerConfig,
) error {
	s.completed = completed
	s.ignored = ignored
	s.clusterConf = clusterConf
	s.kpaConf = kpaConf

	conn, err := sqlite3.Open(dbFileName)
	if err != nil {
		return err
	}
	s.conn = conn

	err = s.conn.Exec(Schema)
	if err != nil {
		return err
	}
	defer s.conn.Close()

	scenarioRunId, err := s.scenarioRun()
	if err != nil {
		return err
	}

	err = s.scenarioData(scenarioRunId)
	if err != nil {
		return err
	}

	return nil
}

func (s *storer) scenarioRun() (scenarioRunId int64, err error) {
	srStmt, err := s.conn.Prepare(`insert into scenario_runs(
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
		return -1, err
	}

	err = srStmt.Exec(
		time.Now().Format(time.RFC3339),
		"skenario_cli",
		"golang_rand_uniform",
		s.clusterConf.LaunchDelay.Nanoseconds(),
		s.clusterConf.TerminateDelay.Nanoseconds(),
		int(s.clusterConf.NumberOfRequests),
		s.kpaConf.TickInterval.Nanoseconds(),
		s.kpaConf.StableWindow.Nanoseconds(),
		s.kpaConf.PanicWindow.Nanoseconds(),
		s.kpaConf.ScaleToZeroGracePeriod.Nanoseconds(),
		s.kpaConf.TargetConcurrencyDefault,
		s.kpaConf.TargetConcurrencyPercentage,
		s.kpaConf.MaxScaleUpRate,
	)
	if err != nil {
		return -1, err
	}

	lastId := s.conn.LastInsertRowID()

	return lastId, nil
}

func (s *storer) scenarioData(scenarioRunId int64) error {
	entityStmt, err := s.conn.Prepare(`insert into entities(name, kind, scenario_run_id) values (?, ?, ?) on conflict do nothing`)
	if err != nil {
		return err
	}
	defer entityStmt.Close()

	stockStmt, err := s.conn.Prepare(`insert into stocks(name, kind_stocked, scenario_run_id) values (?, ?, ?) on conflict do nothing`)
	if err != nil {
		return err
	}
	defer stockStmt.Close()

	for _, mv := range s.completed {
		from := mv.Movement.From()
		to := mv.Movement.To()

		err = entityStmt.Exec(string(mv.Moved.Name()), string(mv.Moved.Kind()), scenarioRunId)
		if err != nil {
			return err
		}

		err = stockStmt.Exec(string(from.Name()), string(from.KindStocked()), scenarioRunId)
		if err != nil {
			return err
		}

		err = stockStmt.Exec(string(to.Name()), string(to.KindStocked()), scenarioRunId)
		if err != nil {
			return err
		}
	}

	return nil
}
