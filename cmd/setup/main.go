// setup is distributed as mlm-setup and as the Docker setup target.
package main

import (
	"fmt"
	"media_library_manager/internal/setupcli"
	"os"
)

var version = "dev"

func main() {
	if err := setupcli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, version); err != nil {
		fmt.Fprintln(os.Stderr, "Setup failed:", err)
		os.Exit(1)
	}
}
