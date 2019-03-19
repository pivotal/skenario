package main

import (
	"fmt"
	"time"

	"knative-simulator/pkg/model"
	"knative-simulator/pkg/simulator"

	fakes "k8s.io/client-go/kubernetes/fake"
)

func main() {
	startPrep := time.Now()

	begin := time.Unix(0, 0).UTC()
	tenMinutes := 10 * time.Minute

	env := simulator.NewEnvironment(begin, tenMinutes)

	exec1 := model.NewExecutable("exec-1", model.StateCold, env)
	endpoints1 := model.NewReplicaEndpoints("endpoints-1", env, fakes.NewSimpleClientset())
	replica1 := model.NewRevisionReplica("revision-1", exec1, endpoints1, env)
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
