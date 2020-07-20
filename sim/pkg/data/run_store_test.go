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
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

func TestRunStore(t *testing.T) {
	spec.Run(t, "RunStore", testStorer, spec.Report(report.Terminal{}))
}

func testStorer(t *testing.T, describe spec.G, it spec.S) {
	var subject RunStore
	var env simulator.Environment
	var startAt time.Time
	var runFor time.Duration
	var clusterConf model.ClusterConfig
	var kpaConf model.AutoscalerConfig

	it.Before(func() {
		startAt = time.Unix(0, 123456789)
		runFor = 10 * time.Minute
		env = simulator.NewEnvironment(context.Background(), startAt, runFor)

		clusterConf = model.ClusterConfig{
			LaunchDelay:      11 * time.Second,
			TerminateDelay:   22 * time.Second,
			NumberOfRequests: 33,
		}
		kpaConf = model.AutoscalerConfig{
			TickInterval: 11 * time.Second,
		}
	})

	describe("Store()", func() {
		var completed []simulator.CompletedMovement
		var ignored []simulator.IgnoredMovement
		var conn *sqlite3.Conn
		var scenarioRunId int64
		var err error
		var stock1, stock2 simulator.ThroughStock

		it.Before(func() {
			var dir string
			dir, err = os.Getwd()
			require.NoError(t, err)
			dbPath := filepath.Join(dir, "skenario_test.db")

			os.Remove(dbPath)

			conn, err = sqlite3.Open(dbPath)
			assert.NoError(t, err)
			assert.NotNil(t, conn)

			subject = NewRunStore(conn)

			stock1 = simulator.NewThroughStock("stock 1", "test entity")
			stock2 = simulator.NewThroughStock("stock 2", "test entity")
			env.AddToSchedule(simulator.NewMovement("stock 1 -> stock 2", startAt.Add(111*time.Second), stock1, stock2))
			env.AddToSchedule(simulator.NewMovement("stock 1 -> stock 2", startAt.Add(222*time.Second), stock1, stock2))
			env.AddToSchedule(simulator.NewMovement("Ignored", env.HaltTime().Add(10*time.Second), simulator.NewSourceStock("Source", "Entity"), simulator.NewSinkStock("Sink", "Entity")))

			completed, ignored, err = env.Run()
			assert.NoError(t, err)

			scenarioRunId, err = subject.Store(completed, ignored, clusterConf, kpaConf, "test_origin", "test_pattern", 10*time.Minute, env.CPUUtilizations())
			assert.NoError(t, err)
		})

		it("returns the scenario_run ID", func() {
			assert.Equal(t, int64(1), scenarioRunId)
		})

		describe("scenario run metadata", func() {
			var recorded, origin, trafficPattern string
			var count int
			var ranFor int64

			it.Before(func() {
				singleQuery(t, conn, `select recorded, simulated_duration, origin, traffic_pattern from scenario_runs`, &recorded, &ranFor, &origin, &trafficPattern)
				singleQuery(t, conn, `select count(1) from scenario_runs`, &count)
			})

			it("inserts a record", func() {
				assert.Equal(t, 1, count)
			})

			it("records a timestamp", func() {
				assert.Contains(t, recorded, time.Now().Format(time.RFC3339))
			})

			it("sets the simulated_duration as 10 minutes", func() {
				assert.Equal(t, 10*time.Minute, time.Duration(ranFor))
			})

			it("sets the origin as 'test_origin'", func() {
				assert.Equal(t, "test_origin", origin)
			})

			it("sets the traffic pattern as 'test_pattern'", func() {
				assert.Equal(t, "test_pattern", trafficPattern)
			})
		})

		describe("scenario parameters", func() {
			var launchDelay, termDelay, numRequests int
			var tickInterval int

			it.Before(func() {
				singleQuery(t, conn, `
					select cluster_launch_delay
						 , cluster_terminate_delay
						 , cluster_number_of_requests
						 , autoscaler_tick_interval
					from scenario_runs `,
					&launchDelay, &termDelay, &numRequests, &tickInterval,
				)
			})

			it("sets cluster configuration", func() {
				assert.Equal(t, 11000000000, launchDelay)
				assert.Equal(t, 22000000000, termDelay)
				assert.Equal(t, 33, numRequests)
			})

			it("sets autoscaler configuration", func() {
				assert.Equal(t, 11000000000, tickInterval)
			})
		})

		describe("entity records", func() {
			var entityCount int
			var name, kind string

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from entities`, &entityCount)
				singleQuery(t, conn, `select name, kind from entities`, &name, &kind)
			})

			it("inserts a record", func() {
				assert.Equal(t, 1, entityCount)
			})

			it("inserts a name", func() {
				assert.Equal(t, "Scenario", name)
			})

			it("inserts a kind", func() {
				assert.Equal(t, "Scenario", kind)
			})
		})

		describe("stock records", func() {
			var stocksCount int
			var name, kind string
			var numStocksWithEmptySimulation = 3
			var numStocksAddedInTest = 4

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from stocks`, &stocksCount)
				singleQuery(t, conn, `select name, kind_stocked from stocks`, &name, &kind)
			})

			it("inserts a record", func() {
				assert.Equal(t, numStocksWithEmptySimulation+numStocksAddedInTest, stocksCount)
			})

			it("inserts a name", func() {
				assert.Equal(t, "BeforeScenario", name)
			})

			it("inserts a kind", func() {
				assert.Equal(t, "Scenario", kind)
			})
		})

		describe("movement records", func() {
			var kind string
			var movementsCount, occursAt int
			var moved, fromStock, toStock int
			var numMovementsWithEmptySimulation = 2

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from completed_movements`, &movementsCount)
				singleQuery(t, conn, `select occurs_at, kind, moved, from_stock, to_stock from completed_movements`, &occursAt, &kind, &moved, &fromStock, &toStock)
			})

			it("inserts a record", func() {
				assert.Equal(t, numMovementsWithEmptySimulation, movementsCount)
			})

			it("inserts the occurrence time", func() {
				assert.Equal(t, 123456789, occursAt)
			})

			it("inserts the movement kind", func() {
				assert.Equal(t, "start_to_running", kind)
			})

			it("inserts the moved entity", func() {
				assert.Equal(t, 1, moved)
			})

			it("inserts the 'from' stock", func() {
				assert.Equal(t, 1, fromStock)
			})

			it("inserts the 'to' stock", func() {
				assert.Equal(t, 2, toStock)
			})

		})

		describe("ignored movement records", func() {
			var ignoredCount, occursAt, fromStock, toStock int
			var kind, reason string

			it.Before(func() {
				singleQuery(t, conn, `select count(1) from ignored_movements`, &ignoredCount)
				singleQuery(t, conn, `select occurs_at, kind, from_stock, to_stock, reason from ignored_movements`, &occursAt, &kind, &fromStock, &toStock, &reason)
			})

			it("inserts a record", func() {
				assert.Equal(t, 3, ignoredCount)
			})

			it("inserts the occurrence time", func() {
				assert.Equal(t, env.HaltTime().Add(10*time.Second).UnixNano(), int64(occursAt))
			})

			it("inserts the movement kind", func() {
				assert.Equal(t, "Ignored", kind)
			})

			it("inserts the 'from' stock", func() {
				assert.Equal(t, 4, fromStock)
			})

			it("inserts the 'to' stock", func() {
				assert.Equal(t, 5, toStock)
			})

			it("inserts a reason for why the movement was ignored", func() {
				assert.Equal(t, "ScheduledToOccurAfterHalt", reason)
			})
		})
	})
}

func singleQuery(t *testing.T, conn *sqlite3.Conn, sql string, scanDst ...interface{}) {
	selectStmt, err := conn.Prepare(sql)
	require.NoError(t, err)

	hasResult, err := selectStmt.Step()
	require.True(t, hasResult)
	require.NoError(t, err)

	err = selectStmt.Scan(scanDst...)
	require.NoError(t, err)

	err = selectStmt.Close()
	require.NoError(t, err)
}
