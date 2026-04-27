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

func timeslipsWriteSubcommands() []*cli.Command {
	return []*cli.Command{
		timeslipsCreateCmd(),
		timeslipsUpdateCmd(),
		timeslipsDeleteCmd(),
		timeslipsTimerStartCmd(),
		timeslipsTimerStopCmd(),
	}
}

func timeslipsCreateCmd() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a timeslip",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "body", Usage: "JSON file with full timeslip payload or timeslip object"},
			&cli.StringFlag{Name: "user", Usage: "User ID or URL (overrides body)"},
			&cli.StringFlag{Name: "project", Usage: "Project ID or URL (overrides body)"},
			&cli.StringFlag{Name: "task", Usage: "Task ID or URL (overrides body)"},
			&cli.StringFlag{Name: "dated-on", Usage: "Date YYYY-MM-DD (overrides body)"},
			&cli.StringFlag{Name: "hours", Usage: "Hours worked, e.g. 1.5 (overrides body)"},
			&cli.StringFlag{Name: "comment", Usage: "Comment (overrides body)"},
		},
		Action: timeslipsCreate,
	}
}

func timeslipsUpdateCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a timeslip",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Timeslip ID"},
			&cli.StringFlag{Name: "url", Usage: "Timeslip URL"},
			&cli.StringFlag{Name: "body", Usage: "JSON file with timeslip payload or timeslip object", Required: true},
		},
		Action: timeslipsUpdate,
	}
}

func timeslipsDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a timeslip",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Timeslip ID"},
			&cli.StringFlag{Name: "url", Usage: "Timeslip URL"},
			&cli.BoolFlag{Name: "yes", Usage: "Skip confirmation prompt"},
		},
		Action: timeslipsDelete,
	}
}

func timeslipsTimerStartCmd() *cli.Command {
	return &cli.Command{
		Name:  "timer-start",
		Usage: "Start the timer on a timeslip",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Timeslip ID"},
			&cli.StringFlag{Name: "url", Usage: "Timeslip URL"},
		},
		Action: timeslipsTimerStart,
	}
}

func timeslipsTimerStopCmd() *cli.Command {
	return &cli.Command{
		Name:  "timer-stop",
		Usage: "Stop the timer on a timeslip",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Timeslip ID"},
			&cli.StringFlag{Name: "url", Usage: "Timeslip URL"},
		},
		Action: timeslipsTimerStop,
	}
}

func timeslipsCreate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}

	timeslip, err := loadResourceObject(c.String("body"), "timeslip")
	if err != nil {
		return err
	}

	for flag, attr := range map[string][2]string{
		"user":    {"users", "user"},
		"project": {"projects", "project"},
		"task":    {"tasks", "task"},
	} {
		if v := strings.TrimSpace(c.String(flag)); v != "" {
			resolved, err := normalizeResourceURL(profile.BaseURL, attr[0], v)
			if err != nil {
				return err
			}
			timeslip[attr[1]] = resolved
		}
	}
	if v := strings.TrimSpace(c.String("dated-on")); v != "" {
		timeslip["dated_on"] = v
	}
	if v := strings.TrimSpace(c.String("hours")); v != "" {
		timeslip["hours"] = v
	}
	if v := strings.TrimSpace(c.String("comment")); v != "" {
		timeslip["comment"] = v
	}

	for _, field := range []string{"user", "project", "task", "dated_on", "hours"} {
		if _, ok := timeslip[field]; !ok {
			return fmt.Errorf("%s is required (set via flag or --body)", field)
		}
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPost, "/timeslips", map[string]any{"timeslip": timeslip})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		created, _ := decoded["timeslip"].(map[string]any)
		if created == nil {
			fmt.Fprintln(os.Stdout, "Timeslip created")
			return nil
		}
		fmt.Fprintf(os.Stdout, "Created timeslip %v\n", created["url"])
		return nil
	})
}

func timeslipsUpdate(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "timeslips", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	timeslip, err := loadResourceObject(c.String("body"), "timeslip")
	if err != nil {
		return err
	}
	if len(timeslip) == 0 {
		return fmt.Errorf("body must contain at least one field to update")
	}

	resp, _, _, err := client.DoJSON(context.Background(), http.MethodPut, path, map[string]any{"timeslip": timeslip})
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Updated timeslip %s\n", path)
		return nil
	})
}

func timeslipsDelete(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "timeslips", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}

	if !c.Bool("yes") && !confirmDelete("timeslip", path) {
		fmt.Fprintln(os.Stdout, "Cancelled")
		return nil
	}

	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Deleted timeslip %s\n", path)
		return nil
	})
}

func timeslipsTimerStart(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "timeslips", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodPost, path+"/timer", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Started timer on %s\n", path)
		return nil
	})
}

func timeslipsTimerStop(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "timeslips", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), http.MethodDelete, path+"/timer", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Stopped timer on %s\n", path)
		return nil
	})
}
