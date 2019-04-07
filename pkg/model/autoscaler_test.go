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

	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/core/v1"
	k8sfakes "k8s.io/client-go/kubernetes/fake"

	"knative-simulator/pkg/simulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "KnativeAutoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

type fakeEnvironment struct {
	movements []simulator.Movement
	listeners []simulator.MovementListener
	theTime   time.Time
}

func (fe *fakeEnvironment) AddToSchedule(movement simulator.Movement) (added bool) {
	fe.movements = append(fe.movements, movement)
	return true
}

func (fe *fakeEnvironment) AddMovementListener(listener simulator.MovementListener) error {
	fe.listeners = append(fe.listeners, listener)
	return nil
}

func (fe *fakeEnvironment) Run() (completed []simulator.CompletedMovement, ignored []simulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func (fe *fakeEnvironment) CurrentMovementTime() time.Time {
	return fe.theTime
}

func (fe *fakeEnvironment) HaltTime() time.Time {
	return fe.theTime.Add(1 * time.Hour)
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

func (fa *fakeAutoscaler) Update(autoscaler.MetricSpec) error {
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
	var envFake *fakeEnvironment
	var cluster ClusterModel
	var config ClusterConfig
	startAt := time.Unix(0, 0)

	it.Before(func() {
		config = ClusterConfig{}
		envFake = &fakeEnvironment{
			movements: make([]simulator.Movement, 0),
			listeners: make([]simulator.MovementListener, 0),
			theTime:   startAt,
		}
		cluster = NewCluster(envFake, config)
	})

	describe("NewKnativeAutoscaler()", func() {
		it.Before(func() {
			subject = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{TickInterval: 60 * time.Second})
			rawSubject = subject.(*knativeAutoscaler)
		})

		describe("scheduling calculations and waits", func() {
			var tickInterval time.Duration
			var calcMovements, waitMovements []simulator.Movement

			it.Before(func() {
				tickInterval = 1 * time.Minute
				calcMovements = []simulator.Movement{}
				waitMovements = []simulator.Movement{}

				for _, mv := range envFake.movements {
					if mv.Kind() == MvWaitingToCalculating {
						calcMovements = append(calcMovements, mv)
					} else if mv.Kind() == MvCalculatingToWaiting {
						waitMovements = append(waitMovements, mv)
					}
				}
			})

			it("schedules an autoscaler_calc movement to occur on each TickInterval", func() {
				assert.Len(t, calcMovements, 59)

				theTime := startAt.Add(1 * time.Nanosecond)
				for _, mv := range calcMovements {
					theTime = theTime.Add(tickInterval)

					assert.Equal(t, theTime, mv.OccursAt())
				}
			})

			it("schedules an autoscaler_wait movement to occur 1ms after each autoscaler_calc", func() {
				assert.Len(t, calcMovements, 59)

				theTime := startAt.Add(1 * time.Nanosecond)
				for _, mv := range waitMovements {
					theTime = theTime.Add(tickInterval)

					assert.Equal(t, theTime.Add(1 * time.Millisecond), mv.OccursAt())
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

		describe("newLogger()", func() {
			var logger *zap.SugaredLogger

			it.Before(func() {
				logger = newLogger()
				assert.NotNil(t, logger)
			})

			it("sets the log level to Info", func() {
				dsl := logger.Desugar()
				assert.True(t, dsl.Core().Enabled(zapcore.InfoLevel))
			})
		})

		describe("newLoggedCtx()", func() {
			var ctx context.Context
			var lg *zap.SugaredLogger

			it.Before(func() {
				lg = newLogger()
				ctx = newLoggedCtx(lg)
			})

			it("has stored the logger in the context", func() {
				assert.Equal(t, lg, logging.FromContext(ctx))
			})
		})

		describe("newKpa() helper", func() {
			var as *autoscaler.Autoscaler
			var conf *autoscaler.Config
			var epiFake *fakeEndpointsInformerSource

			it.Before(func() {
				epiFake = new(fakeEndpointsInformerSource)

				as = newKpa(newLogger(), epiFake, KnativeAutoscalerConfig{
					TickInterval:                11 * time.Second,
					StableWindow:                22 * time.Second,
					PanicWindow:                 33 * time.Second,
					ScaleToZeroGracePeriod:      44 * time.Second,
					TargetConcurrencyDefault:    55.0,
					TargetConcurrencyPercentage: 66.0,
					MaxScaleUpRate:              77.0,
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

			it("sets ContainerCurrencyTargetPercentage", func() {
				assert.Equal(t, 66.0, conf.ContainerConcurrencyTargetPercentage)
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
