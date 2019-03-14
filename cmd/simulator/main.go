package main

import (
	"time"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

func main() {
	begin := time.Unix(0, 0).UTC()
	tenMinutes := 10 * time.Minute

	env := simulator.NewEnvironment(begin, tenMinutes)

	exec1 := model.NewExecutable("exec-1", model.StateCold)
	replica1 := model.NewRevisionReplica("revision-1", exec1, env)
	replica1.Run()
	buffer := model.NewKBuffer(env)
	traffic := model.NewTraffic(env, buffer, replica1, begin, tenMinutes)
	traffic.Run()

	env.Run()
}
