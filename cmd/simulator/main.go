package main

import (
	"time"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment(time.Unix(0, 0), 10*time.Minute)

	model.NewExecutable("exec-1", model.StateCold).Run(env)
	model.NewExecutable("exec-2", model.StateCold).Run(env)
	model.NewExecutable("exec-3", model.StateCold).Run(env)
	model.NewExecutable("exec-4", model.StateDiskWarm).Run(env)
	model.NewExecutable("exec-5", model.StateDiskWarm).Run(env)

	env.Run()
}
