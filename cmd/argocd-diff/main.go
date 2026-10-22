package main

import (
	"fmt"
	"os"

	"github.com/kazysgurskas/argocd-diff/internal/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "argocd-diff:", err)
		os.Exit(1)
	}
}
