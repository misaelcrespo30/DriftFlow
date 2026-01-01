package main

import (
	"fmt"
	//driftflow "github.com/misaelcrespo30/DriftFlow"
	//"github.com/misaelcrespo30/DriftFlow/internal/database/seed"
	"os"

	driftcli "github.com/misaelcrespo30/DriftFlow/cli"
	"github.com/misaelcrespo30/DriftFlow/config"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"
	"github.com/misaelcrespo30/DriftFlow/state"
	"github.com/spf13/cobra"
)

func main() {
	cfg := config.Load()

	// Root para demo
	root := &cobra.Command{Use: "driftflow-demo"}

	// ✅ Modelos fake para probar generate/migrate/etc
	state.SetModels(models.Models())

	//// driftflow.SetSeederRegistry(seed.RegisterSeeders)
	//driftflow.SetSeederRegistry(seed.RegisterSeeders)
	root.AddCommand(driftcli.Commands(cfg)...)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
