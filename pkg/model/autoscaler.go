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
	"context"
	"time"

	"github.com/knative/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"knative-simulator/pkg/simulator"

	"github.com/knative/serving/pkg/autoscaler"
)

const (
	MvWaitingToCalculating simulator.MovementKind = "autoscaler_calc"
	MvCalculatingToWaiting simulator.MovementKind = "autoscaler_wait"

	testNamespace = "simulator-namespace"
	testName      = "revisionService"
)

type KnativeAutoscalerConfig struct {
	TickInterval                time.Duration
	StableWindow                time.Duration
	PanicWindow                 time.Duration
	ScaleToZeroGracePeriod      time.Duration
	TargetConcurrencyDefault    float64
	TargetConcurrencyPercentage float64
	MaxScaleUpRate              float64
}

type KnativeAutoscalerModel interface {
	Model
}

type knativeAutoscaler struct {
	env        simulator.Environment
	tickTock   AutoscalerTicktockStock
}

func (kas *knativeAutoscaler) Env() simulator.Environment {
	return kas.env
}

func NewKnativeAutoscaler(env simulator.Environment, startAt time.Time, cluster ClusterModel, config KnativeAutoscalerConfig) KnativeAutoscalerModel {
	logger := newLogger()
	ctx := newLoggedCtx(logger)
	epiSource := cluster.(EndpointInformerSource)
	kpa := newKpa(logger, epiSource, config)

	autoscalerEntity := simulator.NewEntity("Autoscaler", "Autoscaler")

	kas := &knativeAutoscaler{
		env:        env,
		tickTock:   NewAutoscalerTicktockStock(env, autoscalerEntity, kpa, cluster, ctx),
	}

	for theTime := startAt.Add(config.TickInterval).Add(1 * time.Nanosecond); theTime.Before(env.HaltTime()); theTime = theTime.Add(config.TickInterval) {
		kas.env.AddToSchedule(simulator.NewMovement(
			MvWaitingToCalculating,
			theTime,
			kas.tickTock,
			kas.tickTock,
		))

		kas.env.AddToSchedule(simulator.NewMovement(
			MvCalculatingToWaiting,
			theTime.Add(1*time.Millisecond),
			kas.tickTock,
			kas.tickTock,
		))
	}

	return kas
}

func newLogger() *zap.SugaredLogger {
	devCfg := zap.NewDevelopmentConfig()
	devCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	devCfg.OutputPaths = []string{"stdout"}
	devCfg.ErrorOutputPaths = []string{"stderr"}
	unsugaredLogger, err := devCfg.Build()
	if err != nil {
		panic(err.Error())
	}
	return unsugaredLogger.Sugar()
}

func newLoggedCtx(logger *zap.SugaredLogger) context.Context {
	return logging.WithLogger(context.Background(), logger)
}

func newKpa(logger *zap.SugaredLogger, endpointsInformerSource EndpointInformerSource, kconfig KnativeAutoscalerConfig) *autoscaler.Autoscaler {
	config := &autoscaler.Config{
		TickInterval:                         kconfig.TickInterval,
		MaxScaleUpRate:                       kconfig.MaxScaleUpRate,
		StableWindow:                         kconfig.StableWindow,
		PanicWindow:                          kconfig.PanicWindow,
		ScaleToZeroGracePeriod:               kconfig.ScaleToZeroGracePeriod,
		ContainerConcurrencyTargetPercentage: kconfig.TargetConcurrencyPercentage,
		ContainerConcurrencyTargetDefault:    kconfig.TargetConcurrencyDefault,
	}

	dynConfig := autoscaler.NewDynamicConfig(config, logger)

	statsReporter, err := autoscaler.NewStatsReporter(testNamespace, testName, "config-1", "revision-1")
	if err != nil {
		logger.Fatalf("could not create stats reporter: %s", err.Error())
	}

	as, err := autoscaler.New(
		dynConfig,
		testNamespace,
		testName,
		endpointsInformerSource.EPInformer(),
		kconfig.TargetConcurrencyDefault,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	return as
}
