package newmodel

import (
	"context"
	"testing"
	"time"

	"github.com/knative/serving/pkg/autoscaler"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
)

func TestCluster(t *testing.T) {
	spec.Run(t, "Cluster model", testCluster, spec.Report(report.Terminal{}))
}

func testCluster(t *testing.T, describe spec.G, it spec.S) {
	var subject ClusterModel
	var envFake = new(fakeEnvironment)

	it.Before(func() {
		subject = NewCluster(envFake)
		assert.NotNil(t, subject)
	})

	describe("NewCluster()", func() {
		it("sets an environment", func() {
			assert.Equal(t, envFake, subject.Env())
		})
	})

	describe("CurrentDesired()", func() {
		it("defaults to 0", func() {
			assert.Equal(t, int32(0), subject.CurrentDesired())
		})
	})

	describe("SetDesired()", func() {
		var rawSubject *clusterModel

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			envFake.movements = make([]newsimulator.Movement, 0)

			err := rawSubject.replicasLaunching.Add(newsimulator.NewEntity("already launching", newsimulator.EntityKind("Replica")))
			assert.NoError(t, err)
		})

		describe("new value > ReplicasLaunching.Count()", func() {
			it.Before(func() {
				subject.SetDesired(9)
			})

			it("updates the number of desired replicas", func() {
				assert.Equal(t, int32(9), subject.CurrentDesired())
			})

			it("Adds replica entities to ReplicasLaunching to bring them up to desired", func() {
				assert.Equal(t, uint64(9), rawSubject.replicasLaunching.Count())
			})

			it("schedules movements of new entities from ReplicasLaunching to ReplicasActive", func() {
				assert.Len(t, envFake.movements, 8)
				assert.Equal(t, newsimulator.MovementKind("launching -> active"), envFake.movements[0].Kind())
			})
		})

		describe("new value < ReplicasLaunching.Count()", func() {
			it.Before(func() {
				subject.SetDesired(9)
				subject.SetDesired(0)
			})

			it("updates the number of desired replicas", func() {
				assert.Equal(t, int32(0), subject.CurrentDesired())
			})

			it("Empties ReplicasLaunching", func() {
				assert.Equal(t, uint64(0), rawSubject.replicasLaunching.Count())
			})

			it.Pend("schedules movements from ReplicasActive to ReplicasTerminating", func() {

			})
		})

		describe("new value == ReplicasLaunching.Count()", func() {
			it.Before(func() {
				subject.SetDesired(5)
				subject.SetDesired(5)
			})

			it("doesn't change anything", func() {
				assert.Equal(t, int32(5), subject.CurrentDesired())
				assert.Equal(t, uint64(5), rawSubject.replicasLaunching.Count())
			})
		})
	})

	describe("CurrentLaunching()", func() {
		it.Before(func() {
			subject.SetDesired(7)
		})

		it("gives the .Count() of replicas launching", func() {
			assert.Equal(t, uint64(7), subject.CurrentLaunching())
		})
	})

	describe("CurrentActive()", func() {
		var rawSubject *clusterModel

		it.Before(func() {
			rawSubject = subject.(*clusterModel)
			rawSubject.replicasActive.Add(newsimulator.NewEntity("first entity", "Replica"))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("second entity", "Replica"))
		})

		it("gives the .Count() of replicas active", func() {
			assert.Equal(t, uint64(2), subject.CurrentActive())
		})
	})

	describe("RecordToAutoscaler()", func() {
		var autoscalerFake *fakeAutoscaler
		var rawSubject *clusterModel
		var firstRecorded autoscaler.Stat
		var theTime = time.Now()
		var ctx = context.Background()

		it.Before(func() {
			rawSubject = subject.(*clusterModel)

			autoscalerFake = &fakeAutoscaler{
				recorded:   make([]autoscaler.Stat, 0),
				scaleTimes: make([]time.Time, 0),
			}

			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 1", newsimulator.EntityKind("Replica")))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 2", newsimulator.EntityKind("Replica")))
			rawSubject.replicasActive.Add(newsimulator.NewEntity("Test Replica 3", newsimulator.EntityKind("Replica")))

			subject.RecordToAutoscaler(autoscalerFake, &theTime, ctx)
			firstRecorded = autoscalerFake.recorded[0]
		})

		describe("Records added to the Autoscaler", func() {
			it("records once for each replica in ReplicasActive", func() {
				assert.Len(t, autoscalerFake.recorded, 3)
			})

			it("sets time to the movement OccursAt", func() {
				assert.Equal(t, &theTime, firstRecorded.Time)
			})

			it("sets the PodName to Replica name", func() {
				assert.Equal(t, "Test Replica 1", firstRecorded.PodName)
			})

			it("sets AverageConcurrentRequests to 1", func() {
				assert.Equal(t, float64(1.0), firstRecorded.AverageConcurrentRequests)
			})

			it("sets RequestCount to 1", func() {
				assert.Equal(t, int32(1), firstRecorded.RequestCount)
			})
		})
	})
}
