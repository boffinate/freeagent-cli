package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func apCommand() *cli.Command {
	return &cli.Command{
		Name:  "ap",
		Usage: "Accountancy Practice API",
		Subcommands: []*cli.Command{
			{
				Name:  "practice",
				Usage: "Practice details",
				Subcommands: []*cli.Command{
					apPracticeShowCmd(),
				},
			},
		},
	}
}

func apPracticeShowCmd() *cli.Command {
	return &cli.Command{
		Name:   "show",
		Usage:  "Show the authenticated user's accountancy practice (singleton)",
		Action: apPracticeShow,
	}
}

func apPracticeShow(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/practice", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		// The docs show an unwrapped response ({"name": ..., "subdomain": ...})
		// but tolerate a wrapped {"practice": {...}} variant in case the API
		// normalises later.
		fields := decoded
		if wrapped, ok := decoded["practice"].(map[string]any); ok {
			fields = wrapped
		}
		if fields == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:      %v\n", fields["name"])
		fmt.Fprintf(os.Stdout, "Subdomain: %v\n", fields["subdomain"])
		return nil
	})
}
