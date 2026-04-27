package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

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
			{
				Name:  "account-managers",
				Usage: "Account managers in the practice",
				Subcommands: []*cli.Command{
					apAccountManagersListCmd(),
					apAccountManagersShowCmd(),
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
	resp, _, _, err := client.Do(context.Background(), http.MethodGet, "/practice", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		if decoded == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:      %v\n", decoded["name"])
		fmt.Fprintf(os.Stdout, "Subdomain: %v\n", decoded["subdomain"])
		return nil
	})
}

func apAccountManagersListCmd() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List account managers in the practice",
		Action: apAccountManagersList,
	}
}

func apAccountManagersList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodGet, "/account_managers", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["account_managers"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No account managers found")
			return nil
		}
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "Name\tEmail\tURL")
		for _, item := range list {
			am, ok := item.(map[string]any)
			if !ok {
				continue
			}
			fmt.Fprintf(writer, "%v\t%v\t%v\n", am["name"], am["email"], am["url"])
		}
		return writer.Flush()
	})
}

func apAccountManagersShowCmd() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Get a single account manager by ID or URL",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Account manager ID"},
			&cli.StringFlag{Name: "url", Usage: "Account manager URL"},
		},
		Action: apAccountManagersShow,
	}
}

func apAccountManagersShow(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	id := c.String("id")
	urlValue := c.String("url")
	if id == "" && urlValue == "" {
		return fmt.Errorf("id or url required")
	}
	path, err := requireIDOrURL(profile.BaseURL, "account_managers", id, urlValue)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		am, _ := decoded["account_manager"].(map[string]any)
		if am == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:  %v\n", am["name"])
		fmt.Fprintf(os.Stdout, "Email: %v\n", am["email"])
		fmt.Fprintf(os.Stdout, "URL:   %v\n", am["url"])
		return nil
	})
}
