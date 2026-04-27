package cli

import (
	"context"
	"net/url"

	"github.com/urfave/cli/v2"
)

// Domain 4: payroll. The FreeAgent API exposes payroll as read-only payment
// listings keyed by tax year and period, plus mark_as_paid / mark_as_unpaid
// transitions. payroll_profiles is read-only.

func payrollCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "payroll",
		Usage: "Payroll periods and payments",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List payroll periods for a tax year",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "year", Required: true, Usage: "Tax year, e.g. 2025"},
				},
				Action: payrollList,
			},
			{
				Name:  "get",
				Usage: "Get a single payroll period",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "year", Required: true, Usage: "Tax year"},
					&cli.StringFlag{Name: "period", Required: true, Usage: "Period number"},
				},
				Action: payrollGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, payrollWriteSubcommands()...)
	return cmd
}

func payrollProfilesCommand() *cli.Command {
	return &cli.Command{
		Name:  "payroll-profiles",
		Usage: "Payroll profiles",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List payroll profiles for a tax year",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "year", Required: true, Usage: "Tax year"},
					&cli.StringFlag{Name: "user", Usage: "User ID or URL (optional filter)"},
				},
				Action: payrollProfilesList,
			},
		},
	}
}

func payrollList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", "/payroll/"+c.String("year"), nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func payrollGet(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path := "/payroll/" + c.String("year") + "/" + c.String("period")
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func payrollProfilesList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	q := url.Values{}
	if v := c.String("user"); v != "" {
		resolved, err := normalizeResourceURL(profile.BaseURL, "users", v)
		if err != nil {
			return err
		}
		q.Set("user", resolved)
	}
	path := appendQuery("/payroll_profiles/"+c.String("year"), q.Encode())
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}
