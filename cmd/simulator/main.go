package main

import (
	"time"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment(time.Unix(0, 0), 10*time.Minute)

	simulator.NewDummyProc("foo").Run(env)
	simulator.NewDummyProc("bar").Run(env)

	env.Run()
}
