//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func tasksWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		tasksCreateCmd(),
		tasksUpdateCmd(),
		tasksDeleteCmd(),
	}
}

func tasksCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a task under a project",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full task payload or task object"},
			&cli.StringFlag{Name: "project", Usage: "Project ID or URL (required, used as ?project= query)"},
			&cli.StringFlag{Name: "name", Usage: "Task name (overrides body)"},
			&cli.StringFlag{Name: "status", Usage: "Task status (overrides body)"},
			&cli.StringFlag{Name: "billing-rate", Usage: "Billing rate (overrides body)"},
			&cli.StringFlag{Name: "billing-period", Usage: "Billing period: hour or day (overrides body)"},
			&cli.BoolFlag{Name: "is-billable", Usage: "Mark task as billable (overrides body)"},
		},
		Action: tasksCreate,
	}
}

func tasksUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a task",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Task ID"},
			&cli.StringFlag{Name: "url", Usage: "Task URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with task payload or task object", Required: true},
		},
		Action: tasksUpdate,
	}
}

func tasksDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a task",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Task ID"},
			&cli.StringFlag{Name: "url", Usage: "Task URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: tasksDelete,
	}
}

func tasksCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	project := strings.TrimSpace(c.String("project"))
	if project == "" {
		return fmt.Errorf("project is required")
	}
	projectURL, err := normalizeResourceURL(profile.BaseURL, "projects", project)
	if err != nil {
		return err
	}

	task, err := loadResourceObject(c.String("body"), "task")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(c.String("name")); v != "" {
		task["name"] = v
	}
	if v := strings.TrimSpace(c.String("status")); v != "" {
		task["status"] = v
	}
	if v := strings.TrimSpace(c.String("billing-rate")); v != "" {
		task["billing_rate"] = v
	}
	if v := strings.TrimSpace(c.String("billing-period")); v != "" {
		task["billing_period"] = v
	}
	if c.IsSet("is-billable") {
		task["is_billable"] = c.Bool("is-billable")
	}

	if _, ok := task["name"]; !ok {
		return fmt.Errorf("name is required (set via flag or --body)")
	}

	path := "/tasks?project=" + url.QueryEscape(projectURL)
	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, path, map[string]any{"task": task})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["task"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Task created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created task %v (%v)\n", created["name"], created["url"])
		return nil
	})
}

func tasksUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "tasks", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	task, err := loadResourceObject(c.String("body"), "task")
	if err != nil {
		return err
	}
	if len(task) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"task": task})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated task %s\n", path)
		return nil
	})
}

func tasksDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "tasks", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	if !c.Bool("yes") {
		fmt.Fprintf(os.Stdout, "Delete task %s? (y/N): ", path)
		var answer string
		_, _ = fmt.Fscanln(os.Stdin, &answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stdout, "Cancelled")
			return nil
		}
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted task %s\n", path)
		return nil
	})
}
