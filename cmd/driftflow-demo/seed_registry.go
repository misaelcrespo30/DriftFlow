//go:build demo_seed
// +build demo_seed

package main

import (
	driftflow "github.com/misaelcrespo30/DriftFlow"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/seed"
)

func init() {
	registerSeeders = func() {
		driftflow.SetSeederRegistry(seed.RegisterSeeders)
	}
}
