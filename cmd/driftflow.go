package main

import (
	"fmt"
	"github.com/misaelcrespo30/DriftFlow/internal/models"
	"github.com/misaelcrespo30/DriftFlow/state"
	"os"

	"github.com/misaelcrespo30/DriftFlow/cli"
)

func main() {
	state.SetModels(models.Models())
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
