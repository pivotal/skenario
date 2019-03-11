package main

import (
	"time"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment(time.Unix(0, 0), 10*time.Minute)

	model.NewExecutable("exec-1").Run(env)
	model.NewExecutable("exec-2").Run(env)
	model.NewExecutable("exec-3").Run(env)
	model.NewExecutable("exec-4").Run(env)
	model.NewExecutable("exec-5").Run(env)

	env.Run()
}
