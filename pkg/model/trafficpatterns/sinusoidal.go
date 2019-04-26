package trafficpatterns

import (
	"math"
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type sinusoidal struct {
	env       simulator.Environment
	amplitude int
	period    time.Duration
	source    model.TrafficSource
	buffer    model.RequestsBufferedStock
}

type SinusoidalConfig struct {
	Amplitude int           `json:"amplitude"`
	Period    time.Duration `json:"period"`
}

func (*sinusoidal) Name() string {
	return "sinusoidal"
}

func (s *sinusoidal) Generate() {
	var t time.Time
	startAt := s.env.CurrentMovementTime()

	for t = startAt; t.Before(s.env.HaltTime()); t = t.Add(1 * time.Second) {
		ampl := float64(s.amplitude)
		perd := float64(s.period.Seconds())
		tsec := float64(t.Second())

		rps := ampl*math.Sin((2.0*math.Pi*tsec)/perd) + ampl
		uniRand := NewUniformRandom(s.env, s.source, s.buffer, UniformConfig{
			NumberOfRequests: int(math.Round(rps)),
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
	}
}

func NewSinusoidal(env simulator.Environment, source model.TrafficSource, buffer model.RequestsBufferedStock, config SinusoidalConfig) Pattern {
	return &sinusoidal{
		env:       env,
		amplitude: config.Amplitude,
		period:    config.Period,
		source:    source,
		buffer:    buffer,
	}
}
