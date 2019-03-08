package simulator

import (
	"time"
)

type Process interface {
	Advance(t time.Time, description string)
	Run(env *Environment)
}
