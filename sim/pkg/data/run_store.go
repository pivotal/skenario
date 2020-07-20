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
	"fmt"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type RunStore interface {
	Store(
		completed []simulator.CompletedMovement,
		ignored []simulator.IgnoredMovement,
		clusterConf model.ClusterConfig,
		asConf model.AutoscalerConfig,
		origin string,
		trafficPattern string,
		ranFor time.Duration,
		cpuUtilizations []*simulator.CPUUtilization,
	) (scenarioRunId int64, err error)
}

type storer struct {
	conn            *sqlite3.Conn
	clusterConf     model.ClusterConfig
	asConf          model.AutoscalerConfig
	completed       []simulator.CompletedMovement
	ignored         []simulator.IgnoredMovement
	origin          string
	trafficPattern  string
	ranFor          time.Duration
	cpuUtilizations []*simulator.CPUUtilization
}

func (s *storer) Store(completed []simulator.CompletedMovement, ignored []simulator.IgnoredMovement,
	clusterConf model.ClusterConfig, asConf model.AutoscalerConfig, origin string, trafficPattern string, ranFor time.Duration,
	cpuUtilizations []*simulator.CPUUtilization) (scenarioRunId int64, err error) {

	s.completed = completed
	s.ignored = ignored
	s.clusterConf = clusterConf
	s.asConf = asConf
	s.origin = origin
	s.trafficPattern = trafficPattern
	s.ranFor = ranFor
	s.cpuUtilizations = cpuUtilizations

	scenarioRunId, err = s.scenarioRun()
	if err != nil {
		return scenarioRunId, err
	}

	err = s.conn.WithTx(func() error {
		return s.scenarioData(scenarioRunId)
	})
	if err != nil {
		return scenarioRunId, err
	}

	return scenarioRunId, nil
}

func (s *storer) scenarioRun() (scenarioRunId int64, err error) {
	srStmt, err := s.conn.Prepare(`insert into scenario_runs(
									   recorded
									 , simulated_duration
									 , origin
									 , traffic_pattern
									 , cluster_launch_delay
									 , cluster_terminate_delay
									 , cluster_number_of_requests
									 , autoscaler_tick_interval)
									values (?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return -1, err
	}

	err = srStmt.Exec(
		time.Now().Format(time.RFC3339),
		s.ranFor.Nanoseconds(),
		s.origin,
		s.trafficPattern,
		s.clusterConf.LaunchDelay.Nanoseconds(),
		s.clusterConf.TerminateDelay.Nanoseconds(),
		int(s.clusterConf.NumberOfRequests),
		s.asConf.TickInterval.Nanoseconds(),
	)
	if err != nil {
		return -1, err
	}

	lastId := s.conn.LastInsertRowID()

	return lastId, nil
}

func (s *storer) scenarioData(scenarioRunId int64) error {
	entityStmt, err := s.conn.Prepare(`insert into entities(name, kind) values (?, ?) on conflict do nothing`)
	if err != nil {
		return err
	}
	defer entityStmt.Close()

	stockStmt, err := s.conn.Prepare(`insert into stocks(name, kind_stocked) values (?, ?) on conflict do nothing`)
	if err != nil {
		return err
	}
	defer stockStmt.Close()

	movementStmt, err := s.conn.Prepare(`insert into completed_movements(
            occurs_at
           , kind
           , moved
           , from_stock
           , to_stock
           , scenario_run_id
        ) values (
              ?
            , ?
            , (select id from entities where name = ? and kind = ?)
            , (select id from stocks where name = ? and kind_stocked = ?)
            , (select id from stocks where name = ? and kind_stocked = ?)
            , ?)
    `)
	if err != nil {
		panic(err.Error())
	}
	defer movementStmt.Close()

	for _, mv := range s.completed {
		from := mv.Movement.From()
		to := mv.Movement.To()

		err = entityStmt.Exec(string(mv.Moved.Name()), string(mv.Moved.Kind()))
		if err != nil {
			return err
		}

		err = stockStmt.Exec(string(from.Name()), string(from.KindStocked()))
		if err != nil {
			return err
		}

		err = stockStmt.Exec(string(to.Name()), string(to.KindStocked()))
		if err != nil {
			return err
		}

		err = movementStmt.Exec(
			mv.Movement.OccursAt().UnixNano(),
			string(mv.Movement.Kind()),
			string(mv.Moved.Name()),
			string(mv.Moved.Kind()),
			string(from.Name()),
			string(from.KindStocked()),
			string(to.Name()),
			string(to.KindStocked()),
			scenarioRunId,
		)
		if err != nil {
			return err
		}
	}

	ignoredStmt, err := s.conn.Prepare(`insert into ignored_movements(
		occurs_at
	  , kind
	  , from_stock
	  , to_stock
	  , reason
	  , scenario_run_id
  ) values (
		 ?
	   , ?
	   , (select id from stocks where name = ? and kind_stocked = ?)
	   , (select id from stocks where name = ? and kind_stocked = ?)
	   , ?
	   , ?)
	`)
	if err != nil {
		panic(err.Error())
	}
	defer ignoredStmt.Close()

	for _, mv := range s.ignored {
		from := mv.Movement.From()
		to := mv.Movement.To()

		err = stockStmt.Exec(string(from.Name()), string(from.KindStocked()))
		if err != nil {
			return err
		}

		err = stockStmt.Exec(string(to.Name()), string(to.KindStocked()))
		if err != nil {
			return err
		}

		err = ignoredStmt.Exec(
			mv.Movement.OccursAt().UnixNano(),
			string(mv.Movement.Kind()),
			string(from.Name()),
			string(from.KindStocked()),
			string(to.Name()),
			string(to.KindStocked()),
			mv.Reason,
			scenarioRunId,
		)
		if err != nil {
			return err
		}
	}

	cpuUtilizationStmt, err := s.conn.Prepare(`insert into cpu_utilizations(
		cpu_utilization
	  , calculated_at
	  , scenario_run_id
  ) values (
		 ?
	   , ?
	   , ?)
	`)

	for _, mv := range s.cpuUtilizations {

		err = cpuUtilizationStmt.Exec(
			mv.CPUUtilization,
			mv.CalculatedAt.UnixNano(),
			scenarioRunId,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewRunStore(conn *sqlite3.Conn) RunStore {
	err := conn.Exec(Schema)
	if err != nil {
		panic(fmt.Errorf("could not apply skenario schema: %s", err.Error()))
	}

	return &storer{
		conn: conn,
	}
}
