package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

func usersCommand() *cli.Command {
	return &cli.Command{
		Name:  "users",
		Usage: "Users",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List users",
				Action: usersList,
			},
			{
				Name:  "get",
				Usage: "Get a user by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "User ID"},
					&cli.StringFlag{Name: "url", Usage: "User URL"},
				},
				Action: usersGet,
			},
			{
				Name:   "me",
				Usage:  "Show the current authenticated user",
				Action: usersMe,
			},
		},
	}
}

func usersList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/users", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		var decoded map[string]any
		if err := json.Unmarshal(resp, &decoded); err != nil {
			return err
		}
		list, _ := decoded["users"].([]any)
		if len(list) == 0 {
			fmt.Fprintln(os.Stdout, "No users found")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Name\tEmail\tRole\tURL")
		for _, item := range list {
			u, _ := item.(map[string]any)
			if u == nil {
				continue
			}
			name := fmt.Sprintf("%v %v", u["first_name"], u["last_name"])
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", name, u["email"], u["role"], u["url"])
		}
		return w.Flush()
	})
}

func usersGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "users", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return printUser(resp)
	})
}

func usersMe(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/users/me", nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return printUser(resp)
	})
}

func printUser(resp []byte) error {
	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}
	u, _ := decoded["user"].(map[string]any)
	if u == nil {
		fmt.Fprintln(os.Stdout, string(resp))
		return nil
	}
	fmt.Fprintf(os.Stdout, "Name:  %v %v\n", u["first_name"], u["last_name"])
	fmt.Fprintf(os.Stdout, "Email: %v\n", u["email"])
	fmt.Fprintf(os.Stdout, "Role:  %v\n", u["role"])
	fmt.Fprintf(os.Stdout, "URL:   %v\n", u["url"])
	return nil
}
