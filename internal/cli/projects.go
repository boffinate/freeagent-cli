package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func projectsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "projects",
		Usage: "Projects",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List projects",
				Flags: withPagination(
					&cli.StringFlag{Name: "contact", Usage: "Contact ID or URL"},
					&cli.StringFlag{Name: "view", Usage: "API view filter (active, completed, cancelled, ...)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				),
				Action: projectsList,
			},
			{
				Name:  "get",
				Usage: "Get a project by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Project ID"},
					&cli.StringFlag{Name: "url", Usage: "Project URL"},
				},
				Action: projectsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, projectsWriteSubcommands()...)
	return cmd
}

func projectsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	params := map[string]string{
		"view":          c.String("view"),
		"updated_since": c.String("updated-since"),
	}
	if contact := c.String("contact"); contact != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "contacts", contact)
		if err != nil {
			return err
		}
		params["contact"] = resolved
	}

	resp, err := listAll(context.Background(), client, "/projects", params, "projects", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["projects"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No projects found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Name\tStatus\tContact\tURL")
		for _, item := range list {
			p, _ := item.(map[string]any)
			if p == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", p["name"], p["status"], p["contact"], p["url"])
		}
		return w.Flush()
	})
}

func projectsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "projects", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		p, _ := decoded["project"].(map[string]any)
		if p == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:     %v\n", p["name"])
		fmt.Fprintf(os.Stdout, "Status:   %v\n", p["status"])
		fmt.Fprintf(os.Stdout, "Contact:  %v\n", p["contact"])
		fmt.Fprintf(os.Stdout, "Currency: %v\n", p["currency"])
		fmt.Fprintf(os.Stdout, "URL:      %v\n", p["url"])
		return nil
	})
}
