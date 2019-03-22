package model

import (
	"knative-simulator/pkg/simulator"
)

type Cluster struct {
}

func (c *Cluster) Identity() simulator.ProcessIdentity {
	return "Cluster"
}

func (c *Cluster) UpdateStock(movement simulator.StockMovementEvent) {
	// do nothing
}
