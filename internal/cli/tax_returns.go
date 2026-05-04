package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

// Domain 3 read surface: list/get over the various tax-return resources.
// Writes (mark_as_filed / mark_as_paid transitions) and the sales_tax_periods
// CRUD live in tax_returns_write.go.

func vatReturnsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "vat-returns",
		Usage: "VAT returns",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List VAT returns", Flags: withPagination(), Action: vatReturnsList},
			{
				Name:  "get",
				Usage: "Get a VAT return by period_ends_on (YYYY-MM-DD) or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
					&cli.StringFlag{Name: "url", Usage: "VAT return URL"},
				},
				Action: vatReturnsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, vatReturnsWriteSubcommands()...)
	return cmd
}

func corporationTaxReturnsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "corporation-tax-returns",
		Usage: "Corporation Tax returns",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List Corporation Tax returns", Flags: withPagination(), Action: corporationTaxReturnsList},
			{
				Name:  "get",
				Usage: "Get a Corporation Tax return by period_ends_on or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "period-ends-on", Usage: "Period end date YYYY-MM-DD"},
					&cli.StringFlag{Name: "url", Usage: "Corporation Tax return URL"},
				},
				Action: corporationTaxReturnsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, corporationTaxReturnsWriteSubcommands()...)
	return cmd
}

func selfAssessmentReturnsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "self-assessment-returns",
		Usage: "Self Assessment / Income Tax returns (per user)",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List Self Assessment returns for a user",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Required: true, Usage: "User ID or URL"},
				},
				Action: selfAssessmentReturnsList,
			},
			{
				Name:  "get",
				Usage: "Get a Self Assessment return for a user + period",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "user", Required: true, Usage: "User ID or URL"},
					&cli.StringFlag{Name: "period-ends-on", Required: true, Usage: "Period end date YYYY-MM-DD"},
				},
				Action: selfAssessmentReturnsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, selfAssessmentReturnsWriteSubcommands()...)
	return cmd
}

func cisBandsCommand() *cli.Command {
	return &cli.Command{
		Name:  "cis-bands",
		Usage: "CIS bands (reference data)",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List CIS bands", Flags: withPagination(), Action: cisBandsList},
		},
	}
}

func salesTaxPeriodsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "sales-tax-periods",
		Usage: "Sales tax periods",
		Subcommands: []*cli.Command{
			{Name: "list", Usage: "List sales tax periods", Flags: withPagination(), Action: salesTaxPeriodsList},
			{
				Name:  "get",
				Usage: "Get a sales tax period by ID or URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Sales tax period ID"},
					&cli.StringFlag{Name: "url", Usage: "Sales tax period URL"},
				},
				Action: salesTaxPeriodsGet,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, salesTaxPeriodsWriteSubcommands()...)
	return cmd
}

func vatReturnsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, "/vat_returns", nil, "vat_returns", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "vat_returns", []string{"period_ends_on", "status", "url"})
	})
}

func vatReturnsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := taxReturnPath(profile.BaseURL, "vat_returns", c.String("period-ends-on"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func corporationTaxReturnsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, "/corporation_tax_returns", nil, "corporation_tax_returns", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "corporation_tax_returns", []string{"period_ends_on", "status", "url"})
	})
}

func corporationTaxReturnsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := taxReturnPath(profile.BaseURL, "corporation_tax_returns", c.String("period-ends-on"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func selfAssessmentReturnsList(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	userPath, err := selfAssessmentBasePath(profile.BaseURL, c.String("user"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", userPath, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func selfAssessmentReturnsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	userPath, err := selfAssessmentBasePath(profile.BaseURL, c.String("user"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", userPath+"/"+c.String("period-ends-on"), nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

func cisBandsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, "/cis_bands", nil, "cis_bands", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "cis_bands", []string{"name", "rate", "url"})
	})
}

func salesTaxPeriodsList(c *cli.Context) error {
	rt, client, _, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	resp, err := listAll(context.Background(), client, "/sales_tax_periods", nil, "sales_tax_periods", paginationOptsFrom(c))
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error {
		return renderListTable(resp, "sales_tax_periods", []string{"period_starts_on", "period_ends_on", "url"})
	})
}

func salesTaxPeriodsGet(c *cli.Context) error {
	rt, client, profile, err := bootstrapClient(c)
	if err != nil {
		return err
	}
	path, err := requireIDOrURL(profile.BaseURL, "sales_tax_periods", c.String("id"), c.String("url"))
	if err != nil {
		return err
	}
	resp, _, _, err := client.Do(context.Background(), "GET", path, nil, "")
	if err != nil {
		return err
	}
	return printOrJSON(rt, resp, func() error { return writeRaw(resp) })
}

// taxReturnPath resolves a (period_ends_on, url) flag pair against a tax-return
// resource keyed by date rather than numeric ID.
func taxReturnPath(baseURL, resource, period, urlValue string) (string, error) {
	if urlValue != "" {
		return urlValue, nil
	}
	if period == "" {
		return "", fmt.Errorf("period-ends-on or url required")
	}
	return normalizeResourceURL(baseURL, resource, period)
}

// selfAssessmentBasePath turns a user flag (id or URL) into the
// /v2/users/:user_id/self_assessment_returns prefix used by the income-tax /
// self-assessment endpoints.
func selfAssessmentBasePath(baseURL, user string) (string, error) {
	userURL, err := normalizeResourceURL(baseURL, "users", user)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(userURL)
	if err != nil {
		return "", err
	}
	return parsed.Path + "/self_assessment_returns", nil
}

// renderListTable decodes a {<wrapper>: [...]} list and prints the requested
// fields in declaration order via tabwriter.
func renderListTable(resp []byte, wrapper string, fields []string) error {
	var decoded map[string]any
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return err
	}
	list, _ := decoded[wrapper].([]any)
	if len(list) == 0 {
		fmt.Fprintf(os.Stdout, "No %s found\n", wrapper)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(fields, "\t"))
	for _, item := range list {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		cells := make([]string, len(fields))
		for i, f := range fields {
			cells[i] = fmt.Sprintf("%v", row[f])
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	return w.Flush()
}
