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
	"knative.dev/serving/pkg/apis/serving"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/resources"
	"go.uber.org/zap"

	"skenario/pkg/simulator"

	"knative.dev/serving/pkg/autoscaler"
)

const (
	testNamespace = "simulator-namespace"
	testName      = "revisionService"
)

type KnativeAutoscalerConfig struct {
	TickInterval           time.Duration
	StableWindow           time.Duration
	PanicWindow            time.Duration
	PanicThreshold         float64
	ScaleToZeroGracePeriod time.Duration
	TargetConcurrency      float64
	MaxScaleUpRate         float64
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

	collector, scraper := NewMetricsComponents(logger, config, cluster)
	kpa := newKpa(logger, config, cluster, collector)

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

	scraperTickTock := NewScraperTicktockStock(collector, scraper)
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

func newKpa(logger *zap.SugaredLogger, kconfig KnativeAutoscalerConfig, cluster ClusterModel, collector *autoscaler.MetricCollector) *autoscaler.Autoscaler {
	deciderSpec := autoscaler.DeciderSpec{
		ServiceName:       testName,
		TickInterval:      kconfig.TickInterval,
		MaxScaleUpRate:    kconfig.MaxScaleUpRate,
		TargetConcurrency: kconfig.TargetConcurrency,
		PanicThreshold:    kconfig.PanicThreshold,
		StableWindow:      kconfig.StableWindow,
	}

	statsReporter, err := autoscaler.NewStatsReporter(testNamespace, testName, "config-1", "revision-1")
	if err != nil {
		logger.Fatalf("could not create stats reporter: %s", err.Error())
	}

	clusterAsReadyPods := cluster.(resources.ReadyPodCounter)

	as, err := autoscaler.New(
		testNamespace,
		testName,
		collector,
		clusterAsReadyPods,
		deciderSpec,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	return as
}

func NewMetricsComponents(logger *zap.SugaredLogger, kconfig KnativeAutoscalerConfig, cluster ClusterModel) (*autoscaler.MetricCollector, *autoscaler.ServiceScraper) {
	clusterAsReadyPods := cluster.(resources.ReadyPodCounter)
	clusterAsScrapeClient := cluster.(autoscaler.ScrapeClient)

	metric := &autoscaler.Metric{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Labels:    map[string]string{serving.RevisionLabelKey: testName},
		},
		Spec: autoscaler.MetricSpec{
			StableWindow: kconfig.StableWindow,
			PanicWindow:  kconfig.PanicWindow,
		},
	}
	scraper, err := autoscaler.NewServiceScraper(metric, clusterAsReadyPods, clusterAsScrapeClient)
	if err != nil {
		panic(fmt.Errorf("could not create service scraper: %s", err.Error()))
	}

	clusterStatScraper := func(metric *autoscaler.Metric) (autoscaler.StatsScraper, error) {
		return scraper, nil
	}

	collector := autoscaler.NewMetricCollector(clusterStatScraper, logger)
	_, err = collector.Create(context.Background(), metric)
	if err != nil {
		panic(fmt.Errorf("could not create metric collector: %s", err.Error()))
	}

	return collector, scraper
}

type foo struct{}

func (*foo) Scrape(url string) (*autoscaler.Stat, error) {
	naow := time.Now()
	return &autoscaler.Stat{
		Time:                             &naow,
		PodName:                          "foo-123",
		AverageConcurrentRequests:        10,
		AverageProxiedConcurrentRequests: 20,
		RequestCount:                     30,
		ProxiedRequestCount:              10,
	}, nil
}
