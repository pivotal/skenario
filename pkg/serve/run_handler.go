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
 *
 */

package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"

	"skenario/pkg/data"
	"skenario/pkg/model"
	"skenario/pkg/model/trafficpatterns"
	"skenario/pkg/simulator"
)

var startAt = time.Unix(0, 0)
var runFor = 600 * time.Second

type totalLine struct {
	OccursAt            int    `json:"occurs_at"`
	MovementKind        string `json:"movement_kind"`
	MovedEntity         string `json:"moved_entity"`
	RequestsBuffered    int    `json:"requests_buffered"`
	RequestsProcessing  int    `json:"requests_processing"`
	RequestsCompleted   int    `json:"requests_completed"`
	ReplicasDesired     int    `json:"replicas_desired"`
	ReplicasLaunching   int    `json:"replicas_launching"`
	ReplicasActive      int    `json:"replicas_active"`
	ReplicasTerminating int    `json:"replicas_terminated"`
}

type responseTime struct {
	ArrivedAt    int64 `json:"arrived_at"`
	CompletedAt  int64 `json:"completed_at"`
	ResponseTime int64 `json:"response_time"`
}

type vegaDataSeries struct {
	TotalLines    []totalLine    `json:"total_lines"`
	ResponseTimes []responseTime `json:"response_times"`
}

func RunHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	env := simulator.NewEnvironment(r.Context(), startAt, runFor)

	clusterConf := model.ClusterConfig{
		LaunchDelay:      5 * time.Second,
		TerminateDelay:   1 * time.Second,
		NumberOfRequests: 1000,
	}

	kpaConf := model.KnativeAutoscalerConfig{
		TickInterval:                2 * time.Second,
		StableWindow:                60 * time.Second,
		PanicWindow:                 6 * time.Second,
		ScaleToZeroGracePeriod:      30 * time.Second,
		TargetConcurrencyDefault:    1,
		TargetConcurrencyPercentage: 0.5,
		MaxScaleUpRate:              100,
	}

	cluster := model.NewCluster(env, clusterConf)
	model.NewKnativeAutoscaler(env, startAt, cluster, kpaConf)
	trafficSource := model.NewTrafficSource(env, cluster.BufferStock())
	traffic := trafficpatterns.NewRamp(env, trafficSource, cluster.BufferStock(), 1, 100)
	traffic.Generate()

	completed, ignored, err := env.Run()
	if err != nil {
		panic(err.Error())
	}

	store := data.NewRunStore()
	scenarioRunId, err := store.Store("skenario.db", completed, ignored, clusterConf, kpaConf, "skenario_web", traffic.Name())
	if err != nil {
		fmt.Printf("there was an error saving data: %s", err.Error())
	}

	conn, err := sqlite3.Open("skenario.db")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

	}
	defer conn.Close()

	totalStmt, err := conn.Prepare(data.RunningCountQuery, scenarioRunId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseStmt, err := conn.Prepare(data.ResponseTimesQuery, scenarioRunId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var occursAt, requestsBuffered, requestsProcessing, requestsCompleted, replicasDesired, replicasLaunching, replicasActive, replicasTerminated int
	var arrivedAt, completedAt, rTime int64
	var kind, moved string
	var vds = vegaDataSeries{
		TotalLines:    make([]totalLine, 0),
		ResponseTimes: make([]responseTime, 0),
	}

	for {
		hasRow, err := totalStmt.Step()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !hasRow {
			break
		}

		err = totalStmt.Scan(&occursAt, &kind, &moved, &requestsBuffered, &requestsProcessing, &requestsCompleted, &replicasDesired, &replicasLaunching, &replicasActive, &replicasTerminated)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		line := totalLine{
			OccursAt:            occursAt,
			MovementKind:        kind,
			MovedEntity:         moved,
			RequestsBuffered:    requestsBuffered,
			RequestsProcessing:  requestsProcessing,
			RequestsCompleted:   requestsCompleted,
			ReplicasDesired:     replicasDesired,
			ReplicasLaunching:   replicasLaunching,
			ReplicasActive:      replicasActive,
			ReplicasTerminating: replicasTerminated,
		}
		vds.TotalLines = append(vds.TotalLines, line)
	}

	for {
		hasRow, err := responseStmt.Step()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !hasRow {
			break
		}

		err = responseStmt.Scan(&arrivedAt, &completedAt, &rTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var rt = responseTime{
			ArrivedAt:    arrivedAt,
			CompletedAt:  completedAt,
			ResponseTime: rTime,
		}
		vds.ResponseTimes = append(vds.ResponseTimes, rt)
	}

	err = json.NewEncoder(w).Encode(vds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
