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

package serve

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"skenario/pkg/simulator"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"

	"skenario/pkg/data"
	"skenario/pkg/model"
	"skenario/pkg/model/trafficpatterns"
)

var startAt = time.Unix(0, 0)

type TallyLine struct {
	OccursAt    int64  `json:"occurs_at"`
	StockName   string `json:"stock_name"`
	KindStocked string `json:"kind_stocked"`
	Tally       int64  `json:"tally"`
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

type CPUUtilizationMetric struct {
	CPUUtilization float64 `json:"cpu_utilization"`
	CalculatedAt   int64   `json:"calculated_at"`
}

type SkenarioRunResponse struct {
	RanFor            time.Duration          `json:"ran_for"`
	TrafficPattern    string                 `json:"traffic_pattern"`
	TallyLines        []TallyLine            `json:"tally_lines"`
	ResponseTimes     []ResponseTime         `json:"response_times"`
	RequestsPerSecond []RPS                  `json:"requests_per_second"`
	CPUUtilizations   []CPUUtilizationMetric `json:"cpu_utilizations"`
}

type SkenarioRunRequest struct {
	RunFor           time.Duration `json:"run_for"`
	TrafficPattern   string        `json:"traffic_pattern"`
	InMemoryDatabase bool          `json:"in_memory_database,omitempty"`

	InitialNumberOfReplicas uint `json:"initial_number_of_replicas"`

	LaunchDelay    time.Duration `json:"launch_delay"`
	TerminateDelay time.Duration `json:"terminate_delay"`
	TickInterval   time.Duration `json:"tick_interval"`

	RequestTimeout       time.Duration `json:"request_timeout_nanos"`
	RequestCPUTimeMillis int           `json:"request_cpu_time_millis"`
	RequestIOTimeMillis  int           `json:"request_io_time_millis"`

	UniformConfig    trafficpatterns.UniformConfig    `json:"uniform_config,omitempty"`
	RampConfig       trafficpatterns.RampConfig       `json:"ramp_config,omitempty"`
	StepConfig       trafficpatterns.StepConfig       `json:"step_config,omitempty"`
	SinusoidalConfig trafficpatterns.SinusoidalConfig `json:"sinusoidal_config,omitempty"`
}

var environmentSequence int32 = 0

func RunHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	runReq := &SkenarioRunRequest{}
	err := json.NewDecoder(r.Body).Decode(runReq)
	if err != nil {
		panic(err.Error())
	}

	env := simulator.NewEnvironment(r.Context(), startAt, runReq.RunFor)

	clusterConf := buildClusterConfig(runReq)
	asConf := buildAutoscalerConfig(runReq)
	replicasConfig := model.ReplicasConfig{
		LaunchDelay:    runReq.LaunchDelay,
		TerminateDelay: runReq.TerminateDelay,
	}

	requestConfig := model.RequestConfig{
		CPUTimeMillis: runReq.RequestCPUTimeMillis,
		IOTimeMillis:  runReq.RequestIOTimeMillis,
		Timeout:       runReq.RequestTimeout,
	}

	cluster := model.NewCluster(env, clusterConf, replicasConfig)

	model.NewAutoscaler(env, startAt, cluster, asConf)
	trafficSource := model.NewTrafficSource(env, cluster.RoutingStock(), requestConfig)

	var traffic trafficpatterns.Pattern
	switch runReq.TrafficPattern {
	case "golang_rand_uniform":
		traffic = trafficpatterns.NewUniformRandom(env, trafficSource, cluster.RoutingStock(), runReq.UniformConfig)
	case "step":
		traffic = trafficpatterns.NewStep(env, trafficSource, cluster.RoutingStock(), runReq.StepConfig)
	case "ramp":
		traffic = trafficpatterns.NewRamp(env, trafficSource, cluster.RoutingStock(), runReq.RampConfig)
	case "sinusoidal":
		traffic = trafficpatterns.NewSinusoidal(env, trafficSource, cluster.RoutingStock(), runReq.SinusoidalConfig)
	}

	traffic.Generate()

	completed, ignored, err := env.Run()
	if err != nil {
		panic(err.Error())
	}

	var dbFileName string
	if runReq.InMemoryDatabase {
		dbFileName = "file::memory:?cache=shared"
	} else {
		dbFileName = "skenario.db"
	}

	conn, err := sqlite3.Open(dbFileName)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer conn.Close()

	store := data.NewRunStore(conn)
	scenarioRunId, err := store.Store(completed, ignored, clusterConf, asConf, "skenario_web", traffic.Name(), runReq.RunFor, env.CPUUtilizations())
	if err != nil {
		fmt.Printf("there was an error saving data: %s", err.Error())
	}

	var vds = SkenarioRunResponse{
		RanFor:            env.HaltTime().Sub(startAt),
		TrafficPattern:    traffic.Name(),
		TallyLines:        tallyLines(dbFileName, scenarioRunId),
		ResponseTimes:     responseTimes(dbFileName, scenarioRunId),
		RequestsPerSecond: requestsPerSecond(dbFileName, scenarioRunId),
		CPUUtilizations:   cpuUtilizations(dbFileName, scenarioRunId),
	}

	err = json.NewEncoder(w).Encode(vds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = env.Plugin().Event(startAt.UnixNano(), proto.EventType_DELETE, &skplug.Autoscaler{})
	if err != nil {
		panic(err)
	}
	log.Printf("Deleted autoscaler.")
}

func cpuUtilizations(dbFileName string, scenarioRunId int64) []CPUUtilizationMetric {
	totalConn, err := sqlite3.Open(dbFileName, sqlite3.OPEN_READONLY)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer totalConn.Close()

	cpuUtilizationStmt, err := totalConn.Prepare(data.CPUUtilizationQuery, scenarioRunId)
	if err != nil {
		panic(fmt.Errorf("could not prepare query: %s", err.Error()))
	}

	cpuUtilizations := make([]CPUUtilizationMetric, 0)

	var cpuUtilization float64
	var calculatedAt int64
	for {
		hasRow, err := cpuUtilizationStmt.Step()
		if err != nil {
			panic(fmt.Errorf("could not step: %s", err.Error()))
		}

		if !hasRow {
			break
		}

		err = cpuUtilizationStmt.Scan(&cpuUtilization, &calculatedAt)
		if err != nil {
			panic(fmt.Errorf("could not scan: %s", err.Error()))
		}

		var metric = CPUUtilizationMetric{
			CPUUtilization: cpuUtilization,
			CalculatedAt:   calculatedAt,
		}
		cpuUtilizations = append(cpuUtilizations, metric)
	}
	return cpuUtilizations
}

func tallyLines(dbFileName string, scenarioRunId int64) []TallyLine {
	totalConn, err := sqlite3.Open(dbFileName, sqlite3.OPEN_READONLY)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer totalConn.Close()

	totalStmt, err := totalConn.Prepare(data.RunningTallyQuery, scenarioRunId, scenarioRunId)
	if err != nil {
		panic(fmt.Errorf("could not prepare query: %s", err.Error()))
	}

	var occursAt, tally int64
	var stockName, kindStocked string
	tallyLines := make([]TallyLine, 0)
	for {
		hasRow, err := totalStmt.Step()
		if err != nil {
			panic(fmt.Errorf("could not step: %s", err.Error()))
		}

		if !hasRow {
			break
		}

		err = totalStmt.Scan(&occursAt, &stockName, &kindStocked, &tally)
		if err != nil {
			panic(fmt.Errorf("could not scan: %s", err.Error()))
		}

		line := TallyLine{
			OccursAt:    occursAt,
			StockName:   stockName,
			KindStocked: kindStocked,
			Tally:       tally,
		}
		tallyLines = append(tallyLines, line)
	}

	return tallyLines
}

func responseTimes(dbFileName string, scenarioRunId int64) []ResponseTime {
	responseConn, err := sqlite3.Open(dbFileName, sqlite3.OPEN_READONLY)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer responseConn.Close()

	responseStmt, err := responseConn.Prepare(data.ResponseTimesQuery, scenarioRunId)
	if err != nil {
		panic(fmt.Errorf("could not prepare query: %s", err.Error()))
	}

	var arrivedAt, completedAt, rTime int64
	responseTimes := make([]ResponseTime, 0)
	for {
		hasRow, err := responseStmt.Step()
		if err != nil {
			panic(fmt.Errorf("could not step: %s", err.Error()))
		}

		if !hasRow {
			break
		}

		err = responseStmt.Scan(&arrivedAt, &completedAt, &rTime)
		if err != nil {
			panic(fmt.Errorf("could not scan: %s", err.Error()))
		}

		var rt = ResponseTime{
			ArrivedAt:    arrivedAt,
			CompletedAt:  completedAt,
			ResponseTime: rTime,
		}
		responseTimes = append(responseTimes, rt)
	}

	return responseTimes
}

func requestsPerSecond(dbFileName string, scenarioRunId int64) []RPS {
	rpsConn, err := sqlite3.Open(dbFileName, sqlite3.OPEN_READONLY)
	if err != nil {
		panic(fmt.Errorf("could not open database file '%s': %s", dbFileName, err.Error()))
	}
	defer rpsConn.Close()

	requestsPerSecondStmt, err := rpsConn.Prepare(data.RequestsPerSecondQuery, scenarioRunId)
	if err != nil {
		panic(fmt.Errorf("could not prepare query: %s", err.Error()))
	}

	var second, requests int64
	requestsPerSecond := make([]RPS, 0)
	for {
		hasRow, err := requestsPerSecondStmt.Step()
		if err != nil {
			panic(fmt.Errorf("could not step: %s", err.Error()))
		}

		if !hasRow {
			break
		}

		err = requestsPerSecondStmt.Scan(&second, &requests)
		if err != nil {
			panic(fmt.Errorf("could not scan: %s", err.Error()))
		}

		var rps = RPS{
			Second:   second,
			Requests: requests,
		}
		requestsPerSecond = append(requestsPerSecond, rps)
	}

	return requestsPerSecond
}

func buildClusterConfig(srr *SkenarioRunRequest) model.ClusterConfig {
	return model.ClusterConfig{
		LaunchDelay:             srr.LaunchDelay,
		TerminateDelay:          srr.TerminateDelay,
		NumberOfRequests:        uint(srr.UniformConfig.NumberOfRequests),
		InitialNumberOfReplicas: srr.InitialNumberOfReplicas,
	}
}

func buildAutoscalerConfig(srr *SkenarioRunRequest) model.AutoscalerConfig {
	return model.AutoscalerConfig{
		TickInterval: srr.TickInterval,
	}
}
