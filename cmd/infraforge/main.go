// Command infraforge is the infrastructure provisioning CLI.
package main

import (
	"os"

	"github.com/infraforge/infraforge/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
