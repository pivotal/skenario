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
	var subject KnativeAutoscaler
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
			subject = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{TickInterval:111*time.Second})
			rawSubject = subject.(*knativeAutoscaler)
		})

		describe("scheduling the first waiting to calculation movement", func() {
			it("is an autoscaler_calc movement", func() {
				assert.Equal(t, simulator.MovementKind("autoscaler_calc"), envFake.movements[0].Kind())
			})

			it("OccursAt is based on KnativeAutoscalerConfig.TickInterval + 1ms", func() {
				assert.Equal(t, startAt.Add(111*time.Second).Add(1*time.Millisecond), envFake.movements[0].OccursAt())
			})

			it("has a note about it being the first calculation", func() {
				assert.Equal(t, "First calculation", envFake.movements[0].Notes()[0])
			})
		})

		it("registers itself as a MovementListener", func() {
			assert.Equal(t, subject, envFake.listeners[0])
		})

		it("sets an Environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})

		it("adds an entity representing the autoscaler to the ticktock stock", func() {
			assert.NotNil(t, rawSubject.tickTock.Remove())
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

	describe("OnMovement()", func() {
		var asMovement simulator.Movement
		var ttStock *tickTock
		var theTime = time.Now()

		describe("When moving from waiting to calculating", func() {
			it.Before(func() {
				subject = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{})
				ttStock = &tickTock{}
				asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)

				err := subject.OnMovement(asMovement)
				assert.NoError(t, err)
			})

			it("schedules a calculating -> waiting movement for 1ms later", func() {
				next := envFake.movements[len(envFake.movements)-1]
				assert.Equal(t, MvCalculatingToWaiting, next.Kind())
				assert.Equal(t, theTime.Add(1*time.Millisecond), next.OccursAt())
			})
		})

		describe("When moving from calculating to waiting", func() {
			it.Before(func() {
				subject = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{TickInterval: 999 * time.Second})
				ttStock = &tickTock{}
				asMovement = simulator.NewMovement(MvCalculatingToWaiting, theTime, ttStock, ttStock)

				err := subject.OnMovement(asMovement)
				assert.NoError(t, err)
			})

			it("schedules a waiting -> calculating movement", func() {
				next := envFake.movements[len(envFake.movements)-1]
				assert.Equal(t, MvWaitingToCalculating, next.Kind())
			})

			it("chooses the OccursAt time based on KnativeAutoscalerConfig.TickInterval", func() {
				next := envFake.movements[len(envFake.movements)-1]
				assert.Equal(t, theTime.Add(999 * time.Second), next.OccursAt())
			})
		})

		describe("driving the actual Autoscaler", func() {
			var autoscalerFake *fakeAutoscaler
			var kpa *knativeAutoscaler

			it.Before(func() {
				autoscalerFake = &fakeAutoscaler{
					recorded:   make([]autoscaler.Stat, 0),
					scaleTimes: make([]time.Time, 0),
				}
				kpa = &knativeAutoscaler{
					env:        envFake,
					tickTock:   ttStock,
					cluster:    cluster,
					autoscaler: autoscalerFake,
				}
			})

			describe("controlling time", func() {
				it.Before(func() {
					asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
					err := kpa.OnMovement(asMovement)
					assert.NoError(t, err)
				})

				it("triggers the autoscaler calculation with the movement's OccursAt time", func() {
					assert.Equal(t, theTime, autoscalerFake.scaleTimes[0])
				})
			})

			describe("updating statistics", func() {
				var rawCluster *clusterModel
				onceForBuffer := 1
				onceForReplica := 1

				it.Before(func() {
					rawCluster = cluster.(*clusterModel)
					newReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "22.22.22.22")
					err := rawCluster.replicasActive.Add(newReplica)
					assert.NoError(t, err)

					asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
					err = kpa.OnMovement(asMovement)
					assert.NoError(t, err)
				})

				it("delegates statistics updating to ClusterModel", func() {
					assert.Len(t, autoscalerFake.recorded, onceForBuffer+onceForReplica)
				})
			})

			describe("the autoscaler was able to make a recommendation", func() {
				var rawCluster *clusterModel

				describe("when the desired scale increases", func() {
					it.Before(func() {
						autoscalerFake.scaleTo = 2

						rawCluster = cluster.(*clusterModel)
						newReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "33.33.33.33")
						err := rawCluster.replicasActive.Add(newReplica)
						assert.NoError(t, err)

						asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
						err = kpa.OnMovement(asMovement)
						assert.NoError(t, err)
					})

					it("sets the current desired on the cluster", func() {
						assert.Equal(t, int32(2), kpa.cluster.CurrentDesired())
					})

					it("adds a note", func() {
						assert.Equal(t, "1 ⇑ 2", asMovement.Notes()[0])
					})
				})

				describe("when the desired scale decreases", func() {
					it.Before(func() {
						autoscalerFake.scaleTo = 1

						rawCluster = cluster.(*clusterModel)
						firstReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "44.44.44.44")
						secondReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "55.55.55.55")
						err := rawCluster.replicasActive.Add(firstReplica)
						assert.NoError(t, err)
						err = rawCluster.replicasActive.Add(secondReplica)
						assert.NoError(t, err)

						asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
						err = kpa.OnMovement(asMovement)
						assert.NoError(t, err)
					})

					it("sets the current desired on the cluster", func() {
						assert.Equal(t, int32(1), kpa.cluster.CurrentDesired())
					})

					it("adds a note", func() {
						assert.Equal(t, "2 ⥥ 1", asMovement.Notes()[0])
					})
				})

				describe("when the desired scale is unchanged", func() {
					var activeBefore uint64

					it.Before(func() {
						autoscalerFake.scaleTo = 1

						rawCluster = cluster.(*clusterModel)
						rawCluster.currentDesired = 1
						newReplica := NewReplicaEntity(envFake, rawCluster.kubernetesClient, rawCluster.endpointsInformer, "11.11.11.11")
						err := rawCluster.replicasActive.Add(newReplica)
						assert.NoError(t, err)

						activeBefore = kpa.cluster.CurrentActive()
						asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
						err = kpa.OnMovement(asMovement)
						assert.NoError(t, err)
					})

					it("does not change the current desired on the cluster", func() {
						assert.Equal(t, activeBefore, kpa.cluster.CurrentActive())
						assert.Equal(t, int32(1), kpa.cluster.CurrentDesired())
					})
				})
			})

			describe("the autoscaler failed to make a recommendation", func() {
				it.Before(func() {
					autoscalerFake.cantDecide = true

					asMovement = simulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
					err := kpa.OnMovement(asMovement)
					assert.NoError(t, err)
				})

				it("notes that there was a problem", func() {
					assert.Equal(t, "autoscaler.Scale() was unsuccessful", asMovement.Notes()[0])
				})
			})
		})
	})

	describe("tickTock stock", func() {
		ttStock := &tickTock{}

		it.Before(func() {
			_ = NewKnativeAutoscaler(envFake, startAt, cluster, KnativeAutoscalerConfig{})
		})

		describe("Name()", func() {
			it("is called 'KnativeAutoscaler Stock'", func() {
				assert.Equal(t, ttStock.Name(), simulator.StockName("Autoscaler ticktock"))
			})
		})

		describe("KindStocked()", func() {
			it("accepts Knative Autoscalers", func() {
				assert.Equal(t, ttStock.KindStocked(), simulator.EntityKind("KnativeAutoscaler"))
			})
		})

		describe("Count()", func() {
			it("always has 1 entity stocked", func() {
				assert.Equal(t, ttStock.Count(), uint64(1))

				err := ttStock.Add(simulator.NewEntity("test entity", simulator.EntityKind("KnativeAutoscaler")))
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Count(), uint64(1))
			})
		})

		describe("Remove()", func() {
			it("gives back the one KnativeAutoscaler", func() {
				entity := simulator.NewEntity("test entity", simulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Remove(), entity)
			})
		})

		describe("Add()", func() {
			it("adds the entity if it's not already set", func() {
				assert.Nil(t, ttStock.asEntity)

				entity := simulator.NewEntity("test entity", simulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.asEntity, entity)
			})
		})
	})
}
