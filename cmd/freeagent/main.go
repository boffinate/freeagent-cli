package main

import (
	"log"
	"os"

	"github.com/boffinate/freeagent-cli/internal/cli"
)

var Version = "dev"

func main() {
	app := cli.NewApp(Version)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
