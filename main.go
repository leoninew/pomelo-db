package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/mingyuan/pomelo-db/cmd"
)

//go:embed config.defaults.yaml
var configDefaults []byte

//go:embed VERSION
var version string

func main() {
	if err := cmd.Execute(configDefaults, strings.TrimSpace(version)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
