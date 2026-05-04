package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func timeslipsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "timeslips",
		Usage: "Timeslips",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List timeslips",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "user", Usage: "User ID or URL"},
					&cli.StringFlag{Name: "project", Usage: "Project ID or URL"},
					&cli.StringFlag{Name: "task", Usage: "Task ID or URL"},
					&cli.StringFlag{Name: "from", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "to", Usage: "End date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "updated-since", Usage: "Updated since (YYYY-MM-DD)"},
				}, paginationFlags()...),
				Action: timeslipsList,
			},
			{
				Name:  "get",
				Usage: "Get a timeslip by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Timeslip ID"},
					&cli.StringFlag{Name: "url", Usage: "Timeslip URL"},
				},
				Action: timeslipsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, timeslipsWriteSubcommands()...)
	return cmd
}

func timeslipsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	params := map[string]string{
		"from_date":     c.String("from"),
		"to_date":       c.String("to"),
		"updated_since": c.String("updated-since"),
	}
	for flagName, key := range map[string]string{"user": "user", "project": "project", "task": "task"} {
		if raw := c.String(flagName); raw != "" {
			resource := map[string]string{"user": "users", "project": "projects", "task": "tasks"}[flagName]
			resolved, err := normalizeResourceURL(profile.BaseURL, resource, raw)
			if err != nil {
				return err
			}
			params[key] = resolved
		}
	}

	// Timeslips require a date range (from/to) or updated_since per the API.
	if params["from_date"] == "" && params["to_date"] == "" && params["updated_since"] == "" {
		return fmt.Errorf("provide --from/--to or --updated-since")
	}

	resp, err := listAll(context.Background(), client, "/timeslips", params, "timeslips", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["timeslips"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No timeslips found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Date\tHours\tProject\tTask\tURL")
		for _, item := range list {
			ts, _ := item.(map[string]any)
			if ts == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
				ts["dated_on"], ts["hours"], ts["project"], ts["task"], ts["url"])
		}
		return w.Flush()
	})
}

func timeslipsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "timeslips", c.String("id"), c.String("url"))
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
		ts, _ := decoded["timeslip"].(map[string]any)
		if ts == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Date:    %v\n", ts["dated_on"])
		fmt.Fprintf(os.Stdout, "Hours:   %v\n", ts["hours"])
		fmt.Fprintf(os.Stdout, "User:    %v\n", ts["user"])
		fmt.Fprintf(os.Stdout, "Project: %v\n", ts["project"])
		fmt.Fprintf(os.Stdout, "Task:    %v\n", ts["task"])
		fmt.Fprintf(os.Stdout, "URL:     %v\n", ts["url"])
		return nil
	})
}
