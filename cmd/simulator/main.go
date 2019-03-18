package main

import (
	"fmt"
	"time"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

func main() {
	startPrep := time.Now()

	begin := time.Unix(0, 0).UTC()
	tenMinutes := 10 * time.Minute

	env := simulator.NewEnvironment(begin, tenMinutes)

	exec1 := model.NewExecutable("exec-1", model.StateCold, env)
	replica1 := model.NewRevisionReplica("revision-1", exec1, env)
	replica1.Run()
	buffer := model.NewKBuffer(env)
	traffic := model.NewTraffic(env, buffer, replica1, begin, tenMinutes)
	traffic.Run()

	fmt.Println("=== BEGIN TRACE ===============================================================================================================================================")
	env.Run()
	fmt.Println("=== END TRACE =================================================================================================================================================")

	endSim := time.Now()

	fmt.Printf("Sim clock duration:  %s\n", tenMinutes)
	fmt.Printf("Real clock duration: %s\n", endSim.Sub(startPrep))
}
