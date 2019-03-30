/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

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
