//go:build readonly

package cli

import "github.com/urfave/cli/v2"

func attachmentsWriteSubcommands() []*cli.Command           { return nil }
func notesWriteSubcommands() []*cli.Command                 { return nil }
func journalSetsWriteSubcommands() []*cli.Command           { return nil }
func accountLocksWriteSubcommands() []*cli.Command          { return nil }
func finalAccountsReportsWriteSubcommands() []*cli.Command  { return nil }
