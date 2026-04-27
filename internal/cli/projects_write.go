//go:build !readonly

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func projectsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		projectsCreateCmd(),
		projectsUpdateCmd(),
		projectsDeleteCmd(),
	}
}

func projectsCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a project",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full project payload or project object"},
			&cli.StringFlag{Name: "contact", Usage: "Contact ID, URL, or name (overrides body)"},
			&cli.StringFlag{Name: "name", Usage: "Project name (overrides body)"},
			&cli.StringFlag{Name: "status", Usage: "Project status: Active, Completed, Cancelled, Hidden (overrides body)"},
			&cli.StringFlag{Name: "currency", Usage: "Currency code (overrides body)"},
			&cli.StringFlag{Name: "budget-units", Usage: "Budget units: Hours, Days, Monetary, None (overrides body)"},
			&cli.StringFlag{Name: "budget", Usage: "Budget value (overrides body)"},
		},
		Action: projectsCreate,
	}
}

func projectsUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a project",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Project ID"},
			&cli.StringFlag{Name: "url", Usage: "Project URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with project payload or project object", Required: true},
		},
		Action: projectsUpdate,
	}
}

func projectsDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a project",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Project ID"},
			&cli.StringFlag{Name: "url", Usage: "Project URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: projectsDelete,
	}
}

func projectsCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	project, err := loadResourceObject(c.String("body"), "project")
	if err != nil {
		return err
	}

	if v := strings.TrimSpace(c.String("contact")); v != "" {
		resolved, err := resolveContactValue(c.Context, client, profile.BaseURL, v)
		if err != nil {
			return err
		}
		project["contact"] = resolved
	}
	if v := strings.TrimSpace(c.String("name")); v != "" {
		project["name"] = v
	}
	if v := strings.TrimSpace(c.String("status")); v != "" {
		project["status"] = v
	}
	if v := strings.TrimSpace(c.String("currency")); v != "" {
		project["currency"] = v
	}
	if v := strings.TrimSpace(c.String("budget-units")); v != "" {
		project["budget_units"] = v
	}
	if v := strings.TrimSpace(c.String("budget")); v != "" {
		project["budget"] = v
	}

	for _, field := range []string{"contact", "name", "status", "currency", "budget_units"} {
		if _, ok := project[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/projects", map[string]any{"project": project})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["project"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Project created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created project %v (%v)\n", created["name"], created["url"])
		return nil
	})
}

func projectsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "projects", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	project, err := loadResourceObject(c.String("body"), "project")
	if err != nil {
		return err
	}
	if len(project) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"project": project})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated project %s\n", path)
		return nil
	})
}

func projectsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "projects", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	if !c.Bool("yes") && !confirmDelete("project", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted project %s\n", path)
		return nil
	})
}
