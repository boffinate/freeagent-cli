//go:build !readonly

package cli

import "github.com/urfave/cli/v2"

func writeCommands() []*cli.Command {
	return []*cli.Command{
		bankCommand(),
		rawCommand(),
	}
}
