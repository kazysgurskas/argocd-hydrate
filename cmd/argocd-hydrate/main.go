package main

import (
	"fmt"
	"os"

	"github.com/kazysgurskas/argocd-hydrate/internal/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "argocd-hydrate:", err)
		os.Exit(1)
	}
}
