//go:build !readonly

package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

// confirmDelete prompts y/N naming label+path and returns true on yes.
func confirmDelete(label, path string) bool {
	fmt.Fprintf(os.Stdout, "Delete %s %s? (y/N): ", label, path)
	var answer string
	_, _ = fmt.Fscanln(os.Stdin, &answer)
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// applyTransition issues `PUT base/<transition>` and prints the standard
// success line. Used by every `<resource> transition` subcommand.
func applyTransition(c *cli.Context, base, transition string) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	transition = strings.TrimSpace(transition)
	path := base + "/" + transition
	resp, _, _, err := client.Do(context.Background(), http.MethodPut, path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		fmt.Fprintf(os.Stdout, "Applied %s on %s\n", transition, path)
		return nil
	})
}
