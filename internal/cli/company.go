package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func companyCommand() *cli.Command {
	return &cli.Command{
		Name:  "company",
		Usage: "Company profile",
		Subcommands: []*cli.Command{
			{
				Name:   "show",
				Usage:  "Show the company profile (singleton)",
				Action: companyShow,
			},
		},
	}
}

func companyShow(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/company", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		co, _ := decoded["company"].(map[string]any)
		if co == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:           %v\n", co["name"])
		fmt.Fprintf(os.Stdout, "Type:           %v\n", co["type"])
		fmt.Fprintf(os.Stdout, "Currency:       %v\n", co["currency"])
		fmt.Fprintf(os.Stdout, "Company Number: %v\n", co["company_registration_number"])
		fmt.Fprintf(os.Stdout, "Subdomain:      %v\n", co["subdomain"])
		fmt.Fprintf(os.Stdout, "URL:            %v\n", co["url"])
		return nil
	})
}
