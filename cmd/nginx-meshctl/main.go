// Package main the main entry point for the cli
package main

import (
	"os"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"
)

var (
	pkgName = "nginx-meshctl"
	version = "0.0.0"
	commit  = "local"
)

func main() {
	cmd := commands.Setup(pkgName, version, commit)
	cmd.SilenceUsage = true

	if cmd.Execute() != nil {
		os.Exit(1)
	}
}
