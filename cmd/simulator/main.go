package main

import (
	"time"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment(time.Unix(0, 0), 10*time.Minute)

	exec1 := model.NewExecutable("exec-1", model.StateCold)
	//exec2 := model.NewExecutable("exec-2", model.StateCold)
	//exec3 := model.NewExecutable("exec-3", model.StateCold)
	//exec4 := model.NewExecutable("exec-4", model.StateDiskWarm)
	//exec5 := model.NewExecutable("exec-5", model.StatePageCacheWarm)

	model.NewRevisionReplica("revision-1", exec1, env).Run()
	//model.NewRevisionReplica("revision-2", exec2, env).Run()
	//model.NewRevisionReplica("revision-3", exec3, env).Run()
	//model.NewRevisionReplica("revision-4", exec4, env).Run()
	//model.NewRevisionReplica("revision-5", exec5, env).Run()

	env.Run()
}
