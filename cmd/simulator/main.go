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
	fakeClient := fakes.NewSimpleClientset()

	env := simulator.NewEnvironment(begin, tenMinutes)

	exec1 := model.NewExecutable("exec-1", model.StateCold, env)
	endpoints1 := model.NewReplicaEndpoints("endpoints-1", env, fakeClient)
	autoscaler1 := model.NewAutoscaler("autoscaler-1", env, exec1, endpoints1, fakeClient)

	//model.NewRevisionReplica("replica-manual", exec1, env).Run()

	buffer := model.NewKBuffer(env, autoscaler1)
	traffic := model.NewTraffic(env, buffer, endpoints1, begin, tenMinutes)
	traffic.Run()

	fmt.Println("=== BEGIN TRACE ===============================================================================================================================================")
	env.Run()
	fmt.Println("=== END TRACE =================================================================================================================================================")

	endSim := time.Now()

	fmt.Printf("Sim clock duration:  %s\n", tenMinutes)
	fmt.Printf("Real clock duration: %s\n", endSim.Sub(startPrep))
}
