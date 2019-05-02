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

type TotalLine struct {
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

type ResponseTime struct {
	ArrivedAt    int64 `json:"arrived_at"`
	CompletedAt  int64 `json:"completed_at"`
	ResponseTime int64 `json:"response_time"`
}

type RPS struct {
	Second   int64 `json:"second"`
	Requests int64 `json:"requests"`
}

type SkenarioRunResponse struct {
	RanFor            time.Duration  `json:"ran_for"`
	TrafficPattern    string         `json:"traffic_pattern"`
	TotalLines        []TotalLine    `json:"total_lines"`
	ResponseTimes     []ResponseTime `json:"response_times"`
	RequestsPerSecond []RPS          `json:"requests_per_second"`
}

type SkenarioRunRequest struct {
	RunFor           time.Duration `json:"run_for"`
	TrafficPattern   string        `json:"traffic_pattern"`
	InMemoryDatabase bool          `json:"in_memory_database,omitempty"`

	LaunchDelay                 time.Duration `json:"launch_delay"`
	TerminateDelay              time.Duration `json:"terminate_delay"`
	TickInterval                time.Duration `json:"tick_interval"`
	StableWindow                time.Duration `json:"stable_window"`
	PanicWindow                 time.Duration `json:"panic_window"`
	ScaleToZeroGracePeriod      time.Duration `json:"scale_to_zero_grace_period"`
	TargetConcurrencyDefault    float64       `json:"target_concurrency_default"`
	TargetConcurrencyPercentage float64       `json:"target_concurrency_percentage"`
	MaxScaleUpRate              float64       `json:"max_scale_up_rate"`

	UniformConfig    trafficpatterns.UniformConfig    `json:"uniform_config,omitempty"`
	RampConfig       trafficpatterns.RampConfig       `json:"ramp_config,omitempty"`
	StepConfig       trafficpatterns.StepConfig       `json:"step_config,omitempty"`
	SinusoidalConfig trafficpatterns.SinusoidalConfig `json:"sinusoidal_config,omitempty"`
}

func RunHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	runReq := &SkenarioRunRequest{}
	err := json.NewDecoder(r.Body).Decode(runReq)
	if err != nil {
		panic(err.Error())
	}

	env := simulator.NewEnvironment(r.Context(), startAt, runReq.RunFor)

	clusterConf := buildClusterConfig(runReq)
	kpaConf := buildKpaConfig(runReq)

	cluster := model.NewCluster(env, clusterConf)
	model.NewKnativeAutoscaler(env, startAt, cluster, kpaConf)
	trafficSource := model.NewTrafficSource(env, cluster.BufferStock())

	var traffic trafficpatterns.Pattern
	switch runReq.TrafficPattern {
	case "golang_rand_uniform":
		traffic = trafficpatterns.NewUniformRandom(env, trafficSource, cluster.BufferStock(), runReq.UniformConfig)
	case "step":
		traffic = trafficpatterns.NewStep(env, trafficSource, cluster.BufferStock(), runReq.StepConfig)
	case "ramp":
		traffic = trafficpatterns.NewRamp(env, trafficSource, cluster.BufferStock(), runReq.RampConfig)
	case "sinusoidal":
		traffic = trafficpatterns.NewSinusoidal(env, trafficSource, cluster.BufferStock(), runReq.SinusoidalConfig)
	}

	traffic.Generate()

	completed, ignored, err := env.Run()
	if err != nil {
		panic(err.Error())
	}

	var dbFileName string
	if runReq.InMemoryDatabase {
		dbFileName = ":memory:"
	} else {
		dbFileName = "skenario.db"
	}

	conn, err := sqlite3.Open(dbFileName)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer conn.Close()

	store := data.NewRunStore(conn)
	scenarioRunId, err := store.Store(completed, ignored, clusterConf, kpaConf, "skenario_web", traffic.Name(), runReq.RunFor)
	if err != nil {
		fmt.Printf("there was an error saving data: %s", err.Error())
	}

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

	requestsPerSecondStmt, err := conn.Prepare(data.RequestsPerSecondQuery, scenarioRunId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var occursAt, requestsBuffered, requestsProcessing, requestsCompleted, replicasDesired, replicasLaunching, replicasActive, replicasTerminated int
	var arrivedAt, completedAt, rTime, second, requests int64
	var kind, moved string
	var vds = SkenarioRunResponse{
		RanFor:            env.HaltTime().Sub(startAt),
		TrafficPattern:    traffic.Name(),
		TotalLines:        make([]TotalLine, 0),
		ResponseTimes:     make([]ResponseTime, 0),
		RequestsPerSecond: make([]RPS, 0),
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

		line := TotalLine{
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

		var rt = ResponseTime{
			ArrivedAt:    arrivedAt,
			CompletedAt:  completedAt,
			ResponseTime: rTime,
		}
		vds.ResponseTimes = append(vds.ResponseTimes, rt)
	}

	for {
		hasRow, err := requestsPerSecondStmt.Step()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !hasRow {
			break
		}

		err = requestsPerSecondStmt.Scan(&second, &requests)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var rps = RPS{
			Second:   second,
			Requests: requests,
		}
		vds.RequestsPerSecond = append(vds.RequestsPerSecond, rps)
	}

	err = json.NewEncoder(w).Encode(vds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func buildClusterConfig(srr *SkenarioRunRequest) model.ClusterConfig {
	return model.ClusterConfig{
		LaunchDelay:      srr.LaunchDelay,
		TerminateDelay:   srr.TerminateDelay,
		NumberOfRequests: uint(srr.UniformConfig.NumberOfRequests),
	}
}

func buildKpaConfig(srr *SkenarioRunRequest) model.KnativeAutoscalerConfig {
	return model.KnativeAutoscalerConfig{
		TickInterval:                srr.TickInterval,
		StableWindow:                srr.StableWindow,
		PanicWindow:                 srr.PanicWindow,
		ScaleToZeroGracePeriod:      srr.ScaleToZeroGracePeriod,
		TargetConcurrencyDefault:    srr.TargetConcurrencyDefault,
		TargetConcurrencyPercentage: srr.TargetConcurrencyPercentage,
		MaxScaleUpRate:              srr.MaxScaleUpRate,
	}
}
