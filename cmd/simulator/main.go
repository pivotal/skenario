package main

import (
	"context"
	"time"

	"knative-simulator/pkg/simulator"
)

func main() {
	env := simulator.NewEnvironment()
	env.AddProcess("foo").Run()
	env.AddProcess("bar").Run()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	env.Run(ctx)
}
