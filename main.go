package main

import (
	"fmt"
	"os"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/cmd"
)

var version = "0.0.1-dev"

func main() {
	if err := cmd.NewPlugin(version); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)

		os.Exit(1)
	}
}
