package main

import (
	"fmt"
	"os"

	"github.com/kazysgurskas/argocd-hydrate/internal/cmd"
)

func main() {
	rootCmd := cmd.New()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %s\n", err.Error())
		os.Exit(1)
	}
}
