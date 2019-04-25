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

func TestSinusoidal(t *testing.T) {
	spec.Run(t, "Sinusoidal traffic pattern", testSinusoidal, spec.Report(report.Terminal{}))
}

func testSinusoidal(t *testing.T, describe spec.G, it spec.S) {
	var subject Pattern
	var config SinusoidalConfig
	var envFake *fakes.FakeEnvironment
	var amplitude int
	var period time.Duration
	var trafficSource model.TrafficSource
	var bufferStock model.RequestsBufferedStock

	it.Before(func() {
		amplitude = 20
		period = 20 * time.Second

		envFake = new(fakes.FakeEnvironment)
		envFake.TheHaltTime = envFake.TheTime.Add(60 * time.Second)

		bufferStock = model.NewRequestsBufferedStock(envFake, model.NewReplicasActiveStock(), simulator.NewSinkStock("Failed", "Request"))
		trafficSource = model.NewTrafficSource(envFake, bufferStock)
		config = SinusoidalConfig{
			Amplitude: amplitude,
			Period:    period,
		}
		subject = NewSinusoidal(envFake, trafficSource, bufferStock, config)
	})

	describe("Name()", func() {
		it("calls itself 'sinusoidal'", func() {
			assert.Equal(t, "sinusoidal", subject.Name())
		})
	})

	// TODO: I am not entirely sure how to test this properly
	describe.Pend("Generate()", func() {
		it.Before(func() {
			subject.Generate()
		})
	})
}
