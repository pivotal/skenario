package main

import (
	"flag"
	"fmt"
	"time"

	"knative-simulator/pkg/model"
	"knative-simulator/pkg/simulator"

	fakes "k8s.io/client-go/kubernetes/fake"
)

var simDuration = flag.Duration("duration", 10*time.Minute, "Duration	 of time to simulate.")

func main() {
	flag.Parse()

	startPrep := time.Now()

	begin := time.Unix(0, 0).UTC()
	fakeClient := fakes.NewSimpleClientset()

	env := simulator.NewEnvironment(begin, *simDuration)

	endpoints1 := model.NewReplicaEndpoints("endpoints-1", env, fakeClient)
	autoscaler1 := model.NewAutoscaler("autoscaler-1", env, endpoints1, fakeClient)

	buffer := model.NewKBuffer(env, autoscaler1)
	traffic := model.NewTraffic(env, buffer, endpoints1, begin, *simDuration)
	traffic.Run()

	fmt.Println("=== BEGIN TRACE ===============================================================================================================================================")
	env.Run()
	fmt.Println("=== END TRACE =================================================================================================================================================")

	endSim := time.Now()

	fmt.Printf("Sim clock duration:  %s\n", *simDuration)
	fmt.Printf("Real clock duration: %s\n", endSim.Sub(startPrep))
}
