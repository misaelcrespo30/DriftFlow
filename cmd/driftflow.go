package main

import (
	"fmt"
	"github.com/misaelcrespo30/DriftFlow/cli"
	"os"
)

func main() {

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
