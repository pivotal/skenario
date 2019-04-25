package trafficpatterns

import (
	"time"

	"skenario/pkg/model"
	"skenario/pkg/simulator"
)

type step struct {
	env       simulator.Environment
	rps       int
	stepAfter time.Duration
	source    model.TrafficSource
	buffer    model.RequestsBufferedStock
}

func (*step) Name() string {
	return "step"
}

func (s *step) Generate() {
	var t time.Time
	startAt := s.env.CurrentMovementTime().Add(s.stepAfter)

	for t = startAt; t.Before(s.env.HaltTime()); t = t.Add(1 * time.Second) {
		uniRand := NewUniformRandom(s.env, s.source, s.buffer, UniformConfig{
			NumberOfRequests: s.rps,
			StartAt:          t,
			RunFor:           time.Second,
		})
		uniRand.Generate()
	}
}

func NewStep(env simulator.Environment, rps int, stepAfter time.Duration, source model.TrafficSource, buffer model.RequestsBufferedStock) Pattern {
	return &step{
		env:       env,
		rps:       rps,
		stepAfter: stepAfter,
		source:    source,
		buffer:    buffer,
	}
}
