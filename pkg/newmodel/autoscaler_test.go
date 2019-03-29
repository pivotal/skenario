package newmodel

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

	"knative-simulator/pkg/newsimulator"
)

func TestAutoscaler(t *testing.T) {
	spec.Run(t, "KnativeAutoscaler model", testAutoscaler, spec.Report(report.Terminal{}))
}

type fakeEnvironment struct {
	movements []newsimulator.Movement
	listeners []newsimulator.MovementListener
	theTime   time.Time
}

func (fe *fakeEnvironment) AddToSchedule(movement newsimulator.Movement) (added bool) {
	fe.movements = append(fe.movements, movement)
	return true
}

func (fe *fakeEnvironment) AddMovementListener(listener newsimulator.MovementListener) error {
	fe.listeners = append(fe.listeners, listener)
	return nil
}

func (fe *fakeEnvironment) Run() (completed []newsimulator.CompletedMovement, ignored []newsimulator.IgnoredMovement, err error) {
	return nil, nil, nil
}

func (fe *fakeEnvironment) CurrentMovementTime() time.Time {
	return fe.theTime
}

type fakeAutoscaler struct {
	scaleTimes []time.Time
	cantDecide bool
}

func (fa *fakeAutoscaler) Record(ctx context.Context, stat autoscaler.Stat) {
	panic("implement me")
}

func (fa *fakeAutoscaler) Scale(ctx context.Context, time time.Time) (int32, bool) {
	if fa.cantDecide {
		return 0, false
	}

	fa.scaleTimes = append(fa.scaleTimes, time)
	return 55, true
}

func (fa *fakeAutoscaler) Update(autoscaler.MetricSpec) error {
	panic("implement me")
}

func testAutoscaler(t *testing.T, describe spec.G, it spec.S) {
	var subject KnativeAutoscaler
	var envFake *fakeEnvironment
	var cluster ClusterModel
	startAt := time.Unix(0, 0)

	it.Before(func() {
		envFake = &fakeEnvironment{
			movements: make([]newsimulator.Movement, 0),
			listeners: make([]newsimulator.MovementListener, 0),
			theTime:   startAt,
		}
		cluster = NewCluster(envFake)
	})

	describe("NewKnativeAutoscaler()", func() {
		it.Before(func() {
			subject = NewKnativeAutoscaler(envFake, startAt, cluster)
		})

		it("schedules a first calculation", func() {
			firstCalc := envFake.movements[0]
			assert.Equal(t, newsimulator.MovementKind(MvWaitingToCalculating), firstCalc.Kind())
		})

		it("registers itself as a MovementListener", func() {
			assert.Equal(t, subject, envFake.listeners[0])
		})

		it("sets an Environment", func() {
			assert.Equal(t, envFake, subject.Env())
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

			it.Before(func() {
				as = newKpa(newLogger())
				assert.NotNil(t, as)

				conf = as.Current()
				assert.NotNil(t, conf)
			})

			it("sets StableWindow", func() {
				assert.Equal(t, 60*time.Second, conf.StableWindow)
			})

			it("sets PanicWindow", func() {
				assert.Equal(t, 6*time.Second, conf.PanicWindow)
			})

			it("sets MaxScaleUpRate", func() {
				assert.Equal(t, 10.0, conf.MaxScaleUpRate)
			})

			it("sets ScaleToZeroGracePeriod", func() {
				assert.Equal(t, 30*time.Second, conf.ScaleToZeroGracePeriod)
			})

			it("sets ContainerCurrencyTargetDefault", func() {
				assert.Equal(t, 2.0, conf.ContainerConcurrencyTargetDefault)
			})

			it("sets ContainerCurrencyTargetPercentage", func() {
				assert.Equal(t, 0.5, conf.ContainerConcurrencyTargetPercentage)
			})

			it.Pend("sets the target concurrency at creation", func() {
				// TODO: How to test? This is a private variable.
				// It can be updated through autoscaler.Update() but doesn't have an obvious getter
			})
		})
	})

	describe("OnMovement()", func() {
		var asMovement newsimulator.Movement
		var ttStock *tickTock
		var theTime = time.Now()

		describe("When moving from waiting to calculating", func() {
			it.Before(func() {
				subject = NewKnativeAutoscaler(envFake, startAt, cluster)
				ttStock = &tickTock{}
				asMovement = newsimulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)

				err := subject.OnMovement(asMovement)
				assert.NoError(t, err)
			})

			it("schedules a calculating -> waiting movement", func() {
				next := envFake.movements[1]
				assert.Equal(t, MvCalculatingToWaiting, next.Kind())
			})
		})

		describe("When moving from calculating to waiting", func() {
			it.Before(func() {
				subject = NewKnativeAutoscaler(envFake, startAt, cluster)
				ttStock = &tickTock{}
				asMovement = newsimulator.NewMovement(MvCalculatingToWaiting, theTime, ttStock, ttStock)

				err := subject.OnMovement(asMovement)
				assert.NoError(t, err)
			})

			it("schedules a waiting -> calculating movement", func() {
				next := envFake.movements[1]
				assert.Equal(t, MvWaitingToCalculating, next.Kind())
			})
		})

		describe("driving the actual Autoscaler", func() {
			var autoscalerFake *fakeAutoscaler
			var kpa *knativeAutoscaler

			it.Before(func() {
				autoscalerFake = new(fakeAutoscaler)
				kpa = &knativeAutoscaler{
					env:             envFake,
					tickTock:        ttStock,
					replicasDesired: newsimulator.NewThroughStock("ReplicasDesired", newsimulator.EntityKind("Replica")),
					cluster:         cluster,
					autoscaler:      autoscalerFake,
					lastDesired:     100,
				}
			})

			describe("controlling time", func() {
				it.Before(func() {
					asMovement = newsimulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
					err := kpa.OnMovement(asMovement)
					assert.NoError(t, err)
				})

				it("triggers the autoscaler calculation with the movement's OccursAt time", func() {
					assert.Equal(t, theTime, autoscalerFake.scaleTimes[0])
				})
			})

			describe("the autoscaler was able to make a recommendation", func() {
				describe("when the desired scale increases", func() {
					it.Before(func() {
						kpa.lastDesired = 10
						asMovement = newsimulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
						err := kpa.OnMovement(asMovement)
						assert.NoError(t, err)
					})

					it("updates the last desired record", func() {
						assert.Equal(t, int32(55), kpa.lastDesired)
					})

					it("sets the current desired on the cluster", func() {
						assert.Equal(t, int32(55), kpa.cluster.CurrentDesired())
					})

					it("adds a note", func() {
						assert.Equal(t, "10 ⇑ 55", asMovement.Notes()[0])
					})
				})

				describe("when the desired scale increases", func() {
					it.Before(func() {
						kpa.lastDesired = 99
						asMovement = newsimulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
						err := kpa.OnMovement(asMovement)
						assert.NoError(t, err)
					})

					it("updates the last desired record", func() {
						assert.Equal(t, int32(55), kpa.lastDesired)
					})

					it("sets the current desired on the cluster", func() {
						assert.Equal(t, int32(55), kpa.cluster.CurrentDesired())
					})

					it("adds a note", func() {
						assert.Equal(t, "99 ⥥ 55", asMovement.Notes()[0])
					})
				})
			})

			describe("the autoscaler failed to make a recommendation", func() {
				it.Before(func() {
					autoscalerFake.cantDecide = true

					asMovement = newsimulator.NewMovement(MvWaitingToCalculating, theTime, ttStock, ttStock)
					err := kpa.OnMovement(asMovement)
					assert.NoError(t, err)
				})

				it("notes that there was a problem", func() {
					assert.Equal(t, "autoscaler.Scale() was unsuccessful", asMovement.Notes()[0])
				})

				it("ignores the 'desired' return value, as it is set to zero", func() {
					assert.Equal(t, int32(100), kpa.lastDesired)
				})
			})
		})
	})

	describe("tickTock stock", func() {
		ttStock := &tickTock{}

		it.Before(func() {
			_ = NewKnativeAutoscaler(envFake, startAt, cluster)
		})

		describe("Name()", func() {
			it("is called 'KnativeAutoscaler Stock'", func() {
				assert.Equal(t, ttStock.Name(), newsimulator.StockName("Autoscaler ticktock"))
			})
		})

		describe("KindStocked()", func() {
			it("accepts Knative Autoscalers", func() {
				assert.Equal(t, ttStock.KindStocked(), newsimulator.EntityKind("KnativeAutoscaler"))
			})
		})

		describe("Count()", func() {
			it("always has 1 entity stocked", func() {
				assert.Equal(t, ttStock.Count(), uint64(1))

				err := ttStock.Add(newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler")))
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Count(), uint64(1))
			})
		})

		describe("Remove()", func() {
			it("gives back the one KnativeAutoscaler", func() {
				entity := newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.Remove(), entity)
			})
		})

		describe("Add()", func() {
			it("adds the entity if it's not already set", func() {
				assert.Nil(t, ttStock.asEntity)

				entity := newsimulator.NewEntity("test entity", newsimulator.EntityKind("KnativeAutoscaler"))
				err := ttStock.Add(entity)
				assert.NoError(t, err)

				assert.Equal(t, ttStock.asEntity, entity)
			})
		})
	})
}
