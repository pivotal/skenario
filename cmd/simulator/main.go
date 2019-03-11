package main

import (
	"time"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment(time.Unix(0, 0), 1000*time.Minute)

	simulator.NewDummyProc("fooFirst", "FOO").Run(env)
	simulator.NewDummyProc("barFirst", "BAR").Run(env)

	env.Run()
}
