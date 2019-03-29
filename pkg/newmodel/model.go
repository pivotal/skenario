package newmodel

import "knative-simulator/pkg/newsimulator"

type Model interface {
	Env() newsimulator.Environment
}
