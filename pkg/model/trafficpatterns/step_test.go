package trafficpatterns

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"skenario/pkg/model"
	"skenario/pkg/model/fakes"
	"skenario/pkg/simulator"
)

func TestStep(t *testing.T) {
	spec.Run(t, "Ramp traffic pattern", testStep, spec.Report(report.Terminal{}))
}

func testStep(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var envFake *fakes.FakeEnvironment
	var trafficSource model.TrafficSource
	var bufferStock model.RequestsBufferedStock

	it.Before(func() {
		envFake = new(fakes.FakeEnvironment)
		envFake.TheHaltTime = envFake.TheTime.Add(20 * time.Second)
		bufferStock = model.NewRequestsBufferedStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"))
		trafficSource = model.NewTrafficSource(envFake, bufferStock)

		subject = NewStepPattern(envFake, 10, 10*time.Second, trafficSource, bufferStock)
	})

	describe("Name()", func() {
		it("calls itself 'Step'", func() {
			assert.Equal(t, "step", subject.Name())
		})
	})

	describe("Generate()", func() {
		it.Before(func() {
			subject.Generate()
		})

		describe("constant RPS", func() {
			it("schedules 10 requests in the first step second", func() {
				for i := 0; i < 10; i++ {
					assert.WithinDuration(t, envFake.TheTime.Add(10500*time.Millisecond), envFake.Movements[i].OccursAt(), 500*time.Millisecond)
				}
			})

			it("schedules 10 requests in the last second of the simulation", func() {
				for i := 90; i < 100; i++ {
					assert.WithinDuration(t, envFake.TheTime.Add(19500*time.Millisecond), envFake.Movements[i].OccursAt(), 500*time.Millisecond)
				}
			})
		})

		describe("stepAfter time", func() {
			var startAt time.Time

			it.Before(func() {
				startAt = envFake.TheTime.Add(10 * time.Second)
			})

			it("does not schedule any requests before stepAfter", func() {
				for _, mv := range envFake.Movements {
					assert.True(t, mv.OccursAt().After(startAt))
				}
			})
		})

		describe("the total number of requests", func() {
			it("generates rps * (haltTime - stepAfter) requests in total", func() {
				assert.Len(t, envFake.Movements, 100)
			})
		})

	})
}
