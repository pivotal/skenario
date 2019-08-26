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
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	"knative.dev/serving/pkg/apis/serving"
	"time"

	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/resources"

	"skenario/pkg/simulator"

	"knative.dev/serving/pkg/autoscaler"
)

const (
	testNamespace = "simulator-namespace"
	testName      = "revisionService"
)

type KnativeAutoscalerSpecific struct {
	ScaleToZeroGracePeriod time.Duration
	PanicWindowPercentage  float64
}

type KnativeAutoscalerConfig struct {
	autoscaler.DeciderSpec

	KnativeAutoscalerSpecific
}

type KnativeAutoscalerModel interface {
	Model
}

type knativeAutoscaler struct {
	env      simulator.Environment
	tickTock AutoscalerTicktockStock
}

func (kas *knativeAutoscaler) Env() simulator.Environment {
	return kas.env
}

func NewKnativeAutoscaler(env simulator.Environment, startAt time.Time, cluster ClusterModel, config KnativeAutoscalerConfig) KnativeAutoscalerModel {
	logger := logging.FromContext(env.Context())

	readyPodCounter := NewClusterReadyCounter(cluster.ActiveStock())
	kpa := newKpa(logger, config, readyPodCounter, cluster.Collector())

	autoscalerEntity := simulator.NewEntity("Autoscaler", "Autoscaler")

	kas := &knativeAutoscaler{
		env:      env,
		tickTock: NewAutoscalerTicktockStock(env, autoscalerEntity, kpa, cluster),
	}

	for theTime := startAt.Add(config.TickInterval).Add(1 * time.Nanosecond); theTime.Before(env.HaltTime()); theTime = theTime.Add(config.TickInterval) {
		kas.env.AddToSchedule(simulator.NewMovement(
			"autoscaler_tick",
			theTime,
			kas.tickTock,
			kas.tickTock,
		))
	}

	scraperTickTock := NewScraperTicktockStock(cluster.Collector(), NewClusterServiceScraper(cluster.ActiveStock()))
	for theTime := startAt.Add(config.TickInterval).Add(1 * time.Nanosecond); theTime.Before(env.HaltTime()); theTime = theTime.Add(config.TickInterval) {
		kas.env.AddToSchedule(simulator.NewMovement(
			"scraper_tick",
			theTime,
			scraperTickTock,
			scraperTickTock,
		))
	}

	return kas
}

func newKpa(logger *zap.SugaredLogger, kconfig KnativeAutoscalerConfig, readyCounter resources.ReadyPodCounter, collector *autoscaler.MetricCollector) *autoscaler.Autoscaler {
	kconfig.ServiceName = testName

	statsReporter, err := autoscaler.NewStatsReporter(testNamespace, testName, "config-1", "revision-1")
	if err != nil {
		logger.Fatalf("could not create stats reporter: %s", err.Error())
	}

	as, err := autoscaler.New(
		testNamespace,
		testName,
		collector,
		readyCounter,
		kconfig.DeciderSpec,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	return as
}

func NewMetricCollector(logger *zap.SugaredLogger, kconfig KnativeAutoscalerConfig, activeStock ReplicasActiveStock) *autoscaler.MetricCollector {
	scraper := NewClusterServiceScraper(activeStock)

	stableWindow := float64(kconfig.StableWindow)
	panicFraction := kconfig.PanicWindowPercentage / 100
	panicWindow := time.Duration(panicFraction * stableWindow)

	metric := &v1alpha1.Metric{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Labels:    map[string]string{serving.RevisionLabelKey: testName},
		},
		Spec: v1alpha1.MetricSpec{
			ScrapeTarget: testName,
			StableWindow: kconfig.StableWindow,
			PanicWindow:  panicWindow,
		},
	}

	clusterStatScraper := func(metric *v1alpha1.Metric) (autoscaler.StatsScraper, error) {
		return scraper, nil
	}

	collector := autoscaler.NewMetricCollector(clusterStatScraper, logger)
	err := collector.CreateOrUpdate(metric)
	if err != nil {
		panic(fmt.Errorf("could not create metric collector: %s", err.Error()))
	}

	return collector
}
