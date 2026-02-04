package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"freeagent-cli/internal/config"

	"github.com/urfave/cli/v2"
)

func rawCommand() *cli.Command {
	return &cli.Command{
		Name:  "raw",
		Usage: "Break-glass raw API call",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "method", Value: "GET"},
			&cli.StringFlag{Name: "path", Usage: "Path like /v2/invoices"},
			&cli.StringFlag{Name: "body", Usage: "JSON file to send"},
		},
		Action: rawAction,
	}
}

func rawAction(c *cli.Context) error {
	rt, err := runtimeFrom(c)
	if err != nil {
		return err
	}

	cfg, _, err := loadConfig(rt)
	if err != nil {
		return err
	}
	profile := ensureProfile(cfg, rt.Profile, rt, config.Profile{})

	client, _, err := newClient(context.Background(), rt, profile)
	if err != nil {
		return err
	}

	path := c.String("path")
	if path == "" {
		return fmt.Errorf("path is required")
	}

	method := c.String("method")
	if method == "" {
		method = http.MethodGet
	}

	var body []byte
	if bodyPath := c.String("body"); bodyPath != "" {
		data, err := os.ReadFile(bodyPath)
		if err != nil {
			return err
		}
		body = data
	}

	var resp []byte
	if body != nil {
		resp, _, _, err = client.Do(context.Background(), method, path, bytes.NewReader(body), "application/json")
	} else {
		resp, _, _, err = client.Do(context.Background(), method, path, nil, "")
	}
	if err != nil {
		return err
	}

	if rt.JSONOutput {
		return writeJSONOutput(resp)
	}

	if json.Valid(resp) {
		var indented bytes.Buffer
		if err := json.Indent(&indented, resp, "", "  "); err == nil {
			_, _ = os.Stdout.Write(indented.Bytes())
			_, _ = os.Stdout.Write([]byte("\n"))
			return nil
		}
	}

	_, _ = os.Stdout.Write(resp)
	_, _ = os.Stdout.Write([]byte("\n"))
	return nil
}
