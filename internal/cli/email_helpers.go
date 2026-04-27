//go:build !readonly

package cli

import (
	"encoding/json"
	"os"

	"github.com/urfave/cli/v2"
)

// buildSendEmailPayload constructs the body for /:id/send_email actions.
// If --body is set, the file's contents are used verbatim. Otherwise the
// per-flag values are wrapped under {wrapper: {email: {...}}}.
func buildSendEmailPayload(c *cli.Context, wrapper string) (any, error) {
	if bodyPath := c.String("body"); bodyPath != "" {
		data, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}

	email := map[string]any{}
	if v := c.String("email-to"); v != "" {
		email["to"] = v
	}
	if v := c.String("cc"); v != "" {
		email["cc"] = v
	}
	if v := c.String("bcc"); v != "" {
		email["bcc"] = v
	}
	if v := c.String("subject"); v != "" {
		email["subject"] = v
	}
	if v := c.String("message"); v != "" {
		email["body"] = v
	}
	return map[string]any{wrapper: map[string]any{"email": email}}, nil
}
