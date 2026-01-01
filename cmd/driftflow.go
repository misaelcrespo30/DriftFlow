package main

import (
	"fmt"
	"os"

	"github.com/misaelcrespo30/DriftFlow/cli"
	driftcli "github.com/misaelcrespo30/DriftFlow/cli"
	"github.com/misaelcrespo30/DriftFlow/config"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"
	"github.com/misaelcrespo30/DriftFlow/state"
	"github.com/spf13/cobra"
)

func main() {

	cfg := config.Load()
	rootCmd := &cobra.Command{Use: "driftflow"}
	state.SetModels(models.Models())
	rootCmd.AddCommand(driftcli.Commands(cfg)...) //se agrega lo necesario

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
