package main

import (
	"log"
	"os"

	"github.com/anjor/freeagent-cli/internal/cli"
)

var Version = "dev"

func main() {
	app := cli.NewApp(Version)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
