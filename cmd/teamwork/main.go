package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joshluedeman/teamwork/cmd/teamwork/cmd"
)

// version is set at build time via ldflags:
//
//	go build -ldflags="-X main.version=v1.0.0" ./cmd/teamwork
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Message != "" {
				fmt.Fprintln(os.Stderr, exitErr.Message)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
