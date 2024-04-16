package main

import (
	"fmt"
	"os"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/cmd"
)

func main() {
	if err := cmd.NewPlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)

		os.Exit(1)
	}
}
