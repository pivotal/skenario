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
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/core/v1"
	k8sfakes "k8s.io/client-go/kubernetes/fake"

	"skenario/pkg/simulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "KnativeAutoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

type fakeAutoscaler struct {
	recorded   []autoscaler.Stat
	scaleTimes []time.Time
	cantDecide bool
	scaleTo    int32
}

func (fa *fakeAutoscaler) Record(ctx context.Context, stat autoscaler.Stat) {
	fa.recorded = append(fa.recorded, stat)
}

func (fa *fakeAutoscaler) Scale(ctx context.Context, time time.Time) (int32, bool) {
	if fa.cantDecide {
		return 0, false
	}

	fa.scaleTimes = append(fa.scaleTimes, time)
	return fa.scaleTo, true
}

func (fa *fakeAutoscaler) Update(autoscaler.DeciderSpec) error {
	panic("implement me")
}

type fakeEndpointsInformerSource struct {
	epInformerCalled bool
}

func (feis *fakeEndpointsInformerSource) EPInformer() v1.EndpointsInformer {
	feis.epInformerCalled = true

	fakeClient := k8sfakes.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)
	return informerFactory.Core().V1().Endpoints()
}

func testAutoscaler(t *testing.T, describe spec.G, it spec.S) {
	var subject KnativeAutoscalerModel
	var rawSubject *knativeAutoscaler
	var envFake *FakeEnvironment
	var cluster ClusterModel
	var config ClusterConfig
	var replicasConfig ReplicasConfig
	startAt := time.Unix(0, 0)

	it.Before(func() {
		config = ClusterConfig{}
		replicasConfig = ReplicasConfig{time.Second, time.Second, 100}
		envFake = &FakeEnvironment{
			Movements:   make([]simulator.Movement, 0),
			TheTime:     startAt,
			TheHaltTime: startAt.Add(1 * time.Hour),
		}
		cluster = NewCluster(envFake, config, replicasConfig)
	})

	describe("NewKnativeAutoscaler()", func() {
		it.Before(func() {
			subject = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{TickInterval: 60 * time.Second})
			rawSubject = subject.(*knativeAutoscaler)
		})

		describe("scheduling calculations and waits", func() {
			var tickInterval time.Duration
			var tickMovements []simulator.Movement

			it.Before(func() {
				tickInterval = 1 * time.Minute
				tickMovements = []simulator.Movement{}

				for _, mv := range envFake.Movements {
					if mv.Kind() == "autoscaler_tick" {
						tickMovements = append(tickMovements, mv)
					}
				}
			})

			it("schedules an autoscaler_tick movement to occur on each TickInterval", func() {
				assert.Len(t, tickMovements, 59)

				theTime := startAt.Add(1 * time.Nanosecond)
				for _, mv := range tickMovements {
					theTime = theTime.Add(tickInterval)

					assert.Equal(t, theTime, mv.OccursAt())
				}
			})
		})

		it("sets an Environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})

		it("sets a ticktock stock", func() {
			assert.NotNil(t, rawSubject.tickTock)
			assert.Equal(t, simulator.StockName("Autoscaler Ticktock"), rawSubject.tickTock.Name())
		})

		describe("newKpa() helper", func() {
			var as *autoscaler.Autoscaler
			var conf *autoscaler.Config
			var epiFake *fakeEndpointsInformerSource

			it.Before(func() {
				epiFake = new(fakeEndpointsInformerSource)

				lg, err := zap.NewDevelopment()
				assert.NoError(t, err)
				as = newKpa(lg.Sugar(), epiFake, KnativeAutoscalerConfig{
					TickInterval:           11 * time.Second,
					StableWindow:           22 * time.Second,
					PanicWindow:            33 * time.Second,
					ScaleToZeroGracePeriod: 44 * time.Second,
					TargetConcurrency:      55.0,
					MaxScaleUpRate:         77.0,
				})
				assert.NotNil(t, as)

				conf = as.Current()
				assert.NotNil(t, conf)
			})

			it("sets TickInterval", func() {
				assert.Equal(t, 11*time.Second, conf.TickInterval)
			})

			it("sets StableWindow", func() {
				assert.Equal(t, 22*time.Second, conf.StableWindow)
			})

			it("sets PanicWindow", func() {
				assert.Equal(t, 33*time.Second, conf.PanicWindow)
			})

			it("sets ScaleToZeroGracePeriod", func() {
				assert.Equal(t, 44*time.Second, conf.ScaleToZeroGracePeriod)
			})

			it("sets ContainerCurrencyTargetDefault", func() {
				assert.Equal(t, 55.0, conf.ContainerConcurrencyTargetDefault)
			})

			it("sets MaxScaleUpRate", func() {
				assert.Equal(t, 77.0, conf.MaxScaleUpRate)
			})

			it("gets the endpoints informer from EndpointsInformerSource", func() {
				assert.True(t, epiFake.epInformerCalled)
			})
		})
	})
}
