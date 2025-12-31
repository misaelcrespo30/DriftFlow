package main

import (
	"fmt"
	"os"

	driftcli "github.com/misaelcrespo30/DriftFlow/cli"
	"github.com/misaelcrespo30/DriftFlow/config"
	"github.com/spf13/cobra"
)

func main() {
	cfg := config.Load()

	root := &cobra.Command{Use: "driftflow"}
	root.AddCommand(driftcli.Commands(cfg)...)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
