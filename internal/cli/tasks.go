package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func tasksCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "tasks",
		Usage: "Project tasks",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List tasks",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project ID or URL"},
				},
				Action: tasksList,
			},
			{
				Name:  "get",
				Usage: "Get a task by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Task ID"},
					&cli.StringFlag{Name: "url", Usage: "Task URL"},
				},
				Action: tasksGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, tasksWriteSubcommands()...)
	return cmd
}

func tasksList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	params := map[string]string{}
	if project := c.String("project"); project != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "projects", project)
		if err != nil {
			return err
		}
		params["project"] = resolved
	}
	path := appendQuery("/tasks", buildQueryParams(params))

	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["tasks"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No tasks found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Name\tStatus\tProject\tURL")
		for _, item := range list {
			task, _ := item.(map[string]any)
			if task == nil {
				continue
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", task["name"], task["status"], task["project"], task["url"])
		}
		return w.Flush()
	})
}

func tasksGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "tasks", c.String("id"), c.String("url"))
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
		task, _ := decoded["task"].(map[string]any)
		if task == nil {
			fmt.Fprintln(os.Stdout, string(resp))
			return nil
		}
		fmt.Fprintf(os.Stdout, "Name:    %v\n", task["name"])
		fmt.Fprintf(os.Stdout, "Status:  %v\n", task["status"])
		fmt.Fprintf(os.Stdout, "Project: %v\n", task["project"])
		fmt.Fprintf(os.Stdout, "URL:     %v\n", task["url"])
		return nil
	})
}
