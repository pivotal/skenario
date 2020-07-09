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
	"log"
	"time"

	"go.uber.org/zap"

	"skenario/pkg/simulator"

	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"github.com/knative/serving/pkg/autoscaler"
)

const (
	testNamespace = "simulator-namespace"
	testName      = "revisionService"
)

type AutoscalerConfig struct {
	TickInterval           time.Duration
	StableWindow           time.Duration
	PanicWindow            time.Duration
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

type stubCluster struct{}

// TODO: actually list running pods.
func (c *stubCluster) ListPods() ([]*skplug.Pod, error) {
	return nil, nil
}

func NewAutoscaler(env simulator.Environment, startAt time.Time, cluster ClusterModel, config AutoscalerConfig) KnativeAutoscalerModel {

	autoscalerEntity := simulator.NewEntity("Autoscaler", "Autoscaler")

	err := env.Plugin().Event(startAt.UnixNano(), proto.EventType_CREATE, &skplug.Autoscaler{
		// TODO: select type and plugin based on the scenario.
		Type: "hpa.v2beta2.autoscaling.k8s.io",
		Yaml: hpaYaml,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("Created autoscaler.")

	// TODO: create initial replicas config.
	// Create the first pod since HPA can't scale from zero.
	cm := cluster.(*clusterModel)
	err = cm.replicasActive.Add(cm.replicaSource.Remove())
	if err != nil {
		panic(err)
	}

	kas := &knativeAutoscaler{
		env:      env,
		tickTock: NewAutoscalerTicktockStock(env, autoscalerEntity, cluster),
	}

	for theTime := startAt.Add(config.TickInterval).Add(1 * time.Nanosecond); theTime.Before(env.HaltTime()); theTime = theTime.Add(config.TickInterval) {
		kas.env.AddToSchedule(simulator.NewMovement(
			"autoscaler_tick",
			theTime,
			kas.tickTock,
			kas.tickTock,
		))
	}

	return kas
}

const hpaYaml = `
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa
  namespace: default
spec:
  maxReplicas: 10
  metrics:
  - resource:
      name: cpu
      target:
        averageUtilization: 50
        type: Utilization
    type: Resource
  minReplicas: 1
  scaleTargetRef:
    apiVersion: extensions/v1beta1
    kind: Deployment
    name: deployment
`

func newKpa(logger *zap.SugaredLogger, endpointsInformerSource EndpointInformerSource, kconfig AutoscalerConfig) *autoscaler.Autoscaler {
	config := &autoscaler.Config{
		TickInterval:                      kconfig.TickInterval,
		MaxScaleUpRate:                    kconfig.MaxScaleUpRate,
		StableWindow:                      kconfig.StableWindow,
		PanicWindow:                       kconfig.PanicWindow,
		ScaleToZeroGracePeriod:            kconfig.ScaleToZeroGracePeriod,
		ContainerConcurrencyTargetDefault: kconfig.TargetConcurrency,
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
		kconfig.TargetConcurrency,
		statsReporter,
	)
	if err != nil {
		panic(err.Error())
	}

	return as
}
