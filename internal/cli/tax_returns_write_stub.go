//go:build readonly

package cli

import "github.com/urfave/cli/v2"

func vatReturnsWriteSubcommands() []*cli.Command            { return nil }
func corporationTaxReturnsWriteSubcommands() []*cli.Command { return nil }
func selfAssessmentReturnsWriteSubcommands() []*cli.Command { return nil }
func salesTaxPeriodsWriteSubcommands() []*cli.Command       { return nil }
