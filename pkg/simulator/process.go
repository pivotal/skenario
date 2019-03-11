package simulator

import "time"

type Process interface {
	Name() string
	Advance(t time.Time, eventName string)  (identifier, outcome string)
	Run(env *Environment)
}
