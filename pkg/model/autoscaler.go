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
	"fmt"
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

type KnativeAutoscaler interface {
	Model
	simulator.MovementListener
}

type knativeAutoscaler struct {
	env        simulator.Environment
	tickTock   *tickTock
	cluster    ClusterModel
	autoscaler autoscaler.UniScaler
	ctx        context.Context
	config     KnativeAutoscalerConfig
}

func (kas *knativeAutoscaler) Env() simulator.Environment {
	return kas.env
}

func (kas *knativeAutoscaler) OnMovement(movement simulator.Movement) error {
	switch movement.Kind() {
	case MvWaitingToCalculating:
		occursAt := movement.OccursAt()
		kas.cluster.RecordToAutoscaler(kas.autoscaler, &occursAt, kas.ctx)

		currentlyActive := int32(kas.cluster.CurrentActive())

		desired, ok := kas.autoscaler.Scale(kas.ctx, movement.OccursAt())
		if !ok {
			movement.AddNote("autoscaler.Scale() was unsuccessful")
		} else {
			if desired > currentlyActive {
				movement.AddNote(fmt.Sprintf("%d ⇑ %d", currentlyActive, desired))

				kas.cluster.SetDesired(desired)
			} else if desired < currentlyActive {
				movement.AddNote(fmt.Sprintf("%d ⥥ %d", currentlyActive, desired))

				kas.cluster.SetDesired(desired)
			}
		}

		kas.env.AddToSchedule(simulator.NewMovement(MvCalculatingToWaiting, movement.OccursAt().Add(1*time.Millisecond), kas.tickTock, kas.tickTock))
	case MvCalculatingToWaiting:
		kas.env.AddToSchedule(simulator.NewMovement(MvWaitingToCalculating, movement.OccursAt().Add(kas.config.TickInterval), kas.tickTock, kas.tickTock))
	}

	return nil
}

func NewKnativeAutoscaler(env simulator.Environment, startAt time.Time, cluster ClusterModel, config KnativeAutoscalerConfig) KnativeAutoscaler {
	logger := newLogger()
	ctx := newLoggedCtx(logger)
	epiSource := cluster.(EndpointInformerSource)
	kpa := newKpa(logger, epiSource, config)

	kas := &knativeAutoscaler{
		env:        env,
		tickTock:   &tickTock{},
		cluster:    cluster,
		autoscaler: kpa,
		ctx:        ctx,
		config:     config,
	}

	kas.tickTock.Add(simulator.NewEntity("Autoscaler", "Autoscaler"))
	firstCalculation := simulator.NewMovement(MvWaitingToCalculating, startAt.Add(config.TickInterval).Add(1*time.Millisecond), kas.tickTock, kas.tickTock)
	firstCalculation.AddNote("First calculation")

	env.AddToSchedule(firstCalculation)
	err := env.AddMovementListener(kas)
	if err != nil {
		panic(err.Error())
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

type tickTock struct {
	asEntity simulator.Entity
}

func (tt *tickTock) Name() simulator.StockName {
	return "Autoscaler ticktock"
}

func (tt *tickTock) KindStocked() simulator.EntityKind {
	return simulator.EntityKind("KnativeAutoscaler")
}

func (tt *tickTock) Count() uint64 {
	return 1
}

func (tt *tickTock) EntitiesInStock() []*simulator.Entity {
	return []*simulator.Entity{&tt.asEntity}
}

func (tt *tickTock) Remove() simulator.Entity {
	return tt.asEntity
}

func (tt *tickTock) Add(entity simulator.Entity) error {
	tt.asEntity = entity

	return nil
}
