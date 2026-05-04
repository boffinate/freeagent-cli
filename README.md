# freeagent

A small CLI for the FreeAgent API, built in Go.

## Features

- OAuth login (local callback or manual paste)
- Keychain-backed token storage with file fallback
- Read and write wrappers for common FreeAgent domains: invoices, contacts,
  bills, expenses, projects, tasks, timeslips, estimates, credit notes,
  bank data, tax returns, journal sets, notes, attachments, account locks,
  reports, and reference data
- Break-glass `raw` command for any FreeAgent endpoint
- JSON output mode for scripting / agents
- Read-only build for AI / scripting use (see "Safety model" below)

## Safety model

This repo produces **two binaries** from one source tree:

| Binary | Who it's for | What it can do |
|---|---|---|
| `freeagent` | Humans at a terminal | Full read + write: create invoices, send email, approve bank transactions, delete drafts, arbitrary `raw` calls. |
| `freeagent-ro` | AI agents, scripts, anywhere accidental writes would be unacceptable | Read-only against FreeAgent business data. |

**"Read-only" means**: no mutation of FreeAgent business data (invoices, bank transactions, contacts, etc.). The RO binary still reads/writes local config, reads/writes tokens in the OS keychain / file fallback, and performs OAuth `POST`s to FreeAgent's `/v2/token_endpoint`. These are OAuth-internal and do not touch your FreeAgent business data.

**Why two binaries rather than a `--write` flag:** a flag can be passed by mistake (or passed deliberately by an LLM reading the README). The write code paths are not compiled into `freeagent-ro` at all — there is no flag to flip and no runtime check to bypass.

Two independent safety layers are enforced in CI:

1. **Command-tree exclusion** via Go build tags. `freeagent-ro` registers
   read-only commands only. It does not register `raw`, `bank approve`, or
   write subcommands such as `contacts create`, `invoices create`,
   `invoices send`, `invoices delete`, `bills create`, `expenses delete`,
   `projects update`, `vat-returns transition`, and similar mutations.
2. **HTTP-client guard**. Under `-tags readonly`, every request must satisfy all of:
   - https scheme (no plaintext HTTP — bearer tokens must not traverse unencrypted),
   - host in `{api.freeagent.com, api.sandbox.freeagent.com, api.github.com}`.
     FreeAgent hosts are used for API/OAuth calls; `api.github.com` is used
     only by `version --check`. This blocks bearer-token exfiltration via
     `--base-url` or server-returned absolute URLs while still allowing the
     explicit update check,
   - method is GET/HEAD, **or** POST to the exact path `/v2/token_endpoint` (OAuth flow).

   This catches the case where a future refactor adds a mutating call inside a read subcommand.

## Install

### Full (read + write) binary

```bash
go install github.com/boffinate/freeagent-cli/cmd/freeagent@latest
```

Or build from source:

```bash
make build   # produces bin/freeagent
```

### Read-only binary

```bash
git clone https://github.com/boffinate/freeagent-cli.git
cd freeagent-cli
make install-ro           # installs to $GOPATH/bin/freeagent-ro
# or, to a custom location:
PREFIX=/usr/local/bin make install-ro
```

Tagged binary releases and update checks are published from
`github.com/boffinate/freeagent-cli`.

`make install-ro` runs the readonly test suite (command-tree assertion + HTTP-guard tests) before copying the binary, so a broken RO build cannot quietly land on disk.

### Claude Code harness allow-list example

Add to `.claude/settings.json` to give Claude Code the RO binary but not the full one:

```json
{
  "permissions": {
    "allow": ["Bash(freeagent-ro:*)"],
    "deny":  ["Bash(freeagent:*)"]
  }
}
```

Anchor the path if multiple binaries named `freeagent` may be on `$PATH` — e.g. `Bash(/usr/local/bin/freeagent-ro:*)`.

## Configure

Create a FreeAgent API application and note the client ID + secret.

Save app credentials:

```bash
./freeagent auth configure \
  --client-id YOUR_ID \
  --client-secret YOUR_SECRET \
  --redirect http://127.0.0.1:8797/callback
```

You can also use env vars:

```bash
export FREEAGENT_CLIENT_ID=...
export FREEAGENT_CLIENT_SECRET=...
export FREEAGENT_REDIRECT_URI=http://127.0.0.1:8797/callback
```

## Login

Local callback (default):

```bash
./freeagent auth login
```

Manual flow:

```bash
./freeagent auth login --manual
```

## Usage

> **Pagination.** `list` commands that hit paginated collection endpoints
> auto-paginate by default — the CLI walks the FreeAgent API's `Link`
> header and merges all pages into one response. Auto-pagination is
> bounded at 50 pages by default; if you hit the cap you'll see a stderr
> warning. Override with `--per-page`, `--page`, `--max-pages`, or disable
> with `--no-paginate`. A few list commands hit fixed-shape sub-resources
> (e.g. `payroll list`, `categories list`) and don't take these flags.
> See [`docs/usage.md`](docs/usage.md#pagination) for details.

Create a draft invoice:

```bash
./freeagent invoices create \
  --contact CONTACT_ID \
  --reference INV-001 \
  --lines ./invoice-lines.json
```

You can also pass a contact name or email and the CLI will resolve it:

```bash
./freeagent invoices create \
  --contact "Acme Ltd" \
  --reference INV-002 \
  --lines ./invoice-lines.json
```

Send an invoice email:

```bash
./freeagent invoices send --id INVOICE_ID --email-to you@company.com
```

Mark as sent (no email):

```bash
./freeagent invoices send --id INVOICE_ID
```

Break-glass request:

```bash
./freeagent raw --method GET --path /v2/invoices
```

Contacts:

```bash
./freeagent contacts list
./freeagent contacts search --query "Acme"
./freeagent contacts get --id CONTACT_ID
./freeagent contacts create --organisation "Acme Ltd" --email accounts@acme.test
./freeagent contacts update --id CONTACT_ID --body ./contact-update.json
./freeagent contacts delete --id CONTACT_ID --yes
```

Bank accounts and transactions (read):

```bash
./freeagent bank accounts list
./freeagent bank accounts get --id BANK_ACCOUNT_ID
./freeagent bank transactions list --bank-account BANK_ACCOUNT_ID --from 2026-01-01 --to 2026-01-31
./freeagent bank explanations list --bank-account BANK_ACCOUNT_ID
./freeagent bank explanations get --id EXPLANATION_ID
```

Reports (raw JSON in both table and --json modes):

```bash
./freeagent reports balance-sheet --as-at 2026-03-31
./freeagent reports profit-and-loss --from 2026-01-01 --to 2026-03-31
./freeagent reports trial-balance --from 2026-01-01 --to 2026-03-31
./freeagent reports cashflow --from 2026-01-01 --to 2026-03-31
```

Reference data (company, users, categories, price list, stock):

```bash
./freeagent company show
./freeagent users list
./freeagent users get --id USER_ID
./freeagent users me
./freeagent categories list
./freeagent price-list-items list
./freeagent price-list-items get --id PRICE_LIST_ITEM_ID
./freeagent stock-items list
./freeagent stock-items get --id STOCK_ITEM_ID
./freeagent email-addresses list
./freeagent payroll-profiles list
```

Projects, tasks, timeslips, estimates:

```bash
./freeagent projects list --view active
./freeagent projects get --id PROJECT_ID
./freeagent tasks list --project PROJECT_ID
./freeagent tasks get --id TASK_ID
./freeagent timeslips list --from 2026-01-01 --to 2026-01-31 --user USER_ID
./freeagent timeslips get --id TIMESLIP_ID
./freeagent estimates list --contact CONTACT_ID
./freeagent estimates get --id ESTIMATE_ID
```

Accountancy Practice (read):

```bash
./freeagent ap practice show
./freeagent ap account-managers list
./freeagent ap account-managers show --id ACCOUNT_MANAGER_ID
./freeagent ap clients list
./freeagent ap clients list --view active --sort -created_at
./freeagent ap clients list --minimal --per-page 500
./freeagent ap clients list --from-date 2024-01-01 --to-date 2024-12-31
./freeagent ap clients list --updated-since 2024-06-01T00:00:00Z
```

The `ap` commands require a token issued for an app with **Accountancy
Practice API** enabled in the FreeAgent Developer Dashboard. A token from
a non-accountant account will surface FreeAgent's API error verbatim
(typically 401/403). There is no client-side gate — the binary lets you
attempt the call so the upstream error is visible.

Acting on behalf of a practice client (any standard endpoint):

```bash
./freeagent --subdomain CLIENT_SUBDOMAIN contacts list
./freeagent --client CLIENT_SUBDOMAIN invoices list
FREEAGENT_SUBDOMAIN=acme ./freeagent reports balance-sheet --as-at 2026-03-31
```

`--subdomain` (alias `--client`, env `FREEAGENT_SUBDOMAIN`) is global: it
adds `X-Subdomain: CLIENT_SUBDOMAIN` to every request, so the existing
read commands work per-client without parallel `ap <command>` mirrors.
The flag also works with `freeagent-ro` for safe per-client reads.

Bills, expenses, credit notes (read):

```bash
./freeagent bills list --from 2026-01-01 --to 2026-03-31
./freeagent bills get --id BILL_ID
./freeagent expenses list --user USER_ID --from 2026-01-01
./freeagent expenses get --id EXPENSE_ID
./freeagent credit-notes list --contact CONTACT_ID
./freeagent credit-notes get --id CREDIT_NOTE_ID
```

Other read-only domains:

```bash
./freeagent account-locks list
./freeagent attachments get --id ATTACHMENT_ID
./freeagent capital-assets list
./freeagent capital-assets get --id CAPITAL_ASSET_ID
./freeagent capital-asset-types list
./freeagent capital-asset-types get --id CAPITAL_ASSET_TYPE_ID
./freeagent hire-purchases list
./freeagent hire-purchases get --id HIRE_PURCHASE_ID
./freeagent journal-sets list
./freeagent journal-sets get --id JOURNAL_SET_ID
./freeagent journal-sets opening-balances
./freeagent notes list --contact CONTACT_ID
./freeagent notes get --id NOTE_ID
./freeagent properties list
./freeagent properties get --id PROPERTY_ID
./freeagent recurring-invoices list
./freeagent recurring-invoices get --id RECURRING_INVOICE_ID
```

Bills (write — full binary only):

```bash
./freeagent bills create --body ./bill.json
./freeagent bills create --contact CONTACT_ID --reference REF100 \
  --dated-on 2026-04-01 --due-on 2026-05-01 --items ./bill-items.json
./freeagent bills update --id BILL_ID --body ./bill-update.json
./freeagent bills delete --id BILL_ID --yes
```

Expenses (write — full binary only):

```bash
./freeagent expenses create --body ./expense.json
./freeagent expenses create --user USER_ID --category 285 \
  --dated-on 2026-04-01 --gross-value 12.50 --description "Train fare"
./freeagent expenses create --user USER_ID --category Mileage \
  --dated-on 2026-04-01
./freeagent expenses update --id EXPENSE_ID --body ./expense-update.json
./freeagent expenses delete --id EXPENSE_ID --yes
```

Projects, tasks, timeslips (write — full binary only):

```bash
./freeagent projects create --contact CONTACT_ID --name "Site rebuild" \
  --status Active --currency GBP --budget-units Hours
./freeagent projects update --id PROJECT_ID --body ./project-update.json
./freeagent projects delete --id PROJECT_ID --yes

./freeagent tasks create --project PROJECT_ID --name "Design"
./freeagent tasks update --id TASK_ID --body ./task-update.json
./freeagent tasks delete --id TASK_ID --yes

./freeagent timeslips create --user USER_ID --project PROJECT_ID \
  --task TASK_ID --dated-on 2026-04-01 --hours 1.5
./freeagent timeslips update --id TIMESLIP_ID --body ./timeslip-update.json
./freeagent timeslips delete --id TIMESLIP_ID --yes
./freeagent timeslips timer-start --id TIMESLIP_ID
./freeagent timeslips timer-stop  --id TIMESLIP_ID
```

Credit notes & estimates (write — full binary only):

```bash
./freeagent credit-notes create --contact CONTACT_ID --reference CN-001 \
  --dated-on 2026-04-01 --currency GBP --items ./credit-note-items.json
./freeagent credit-notes update --id CN_ID --body ./credit-note-update.json
./freeagent credit-notes send --id CN_ID --email-to client@example.com
./freeagent credit-notes transition --id CN_ID --name mark_as_sent
./freeagent credit-notes delete --id CN_ID --yes

./freeagent estimates create --contact CONTACT_ID --reference EST-100 \
  --dated-on 2026-04-01 --currency GBP --items ./estimate-items.json
./freeagent estimates update --id EST_ID --body ./estimate-update.json
./freeagent estimates send --id EST_ID --email-to client@example.com
./freeagent estimates transition --id EST_ID --name mark_as_approved
./freeagent estimates duplicate --id EST_ID
./freeagent estimates delete --id EST_ID --yes
./freeagent estimates items create --estimate EST_ID \
  --description "Design" --price 100 --item-type Services
./freeagent estimates items update --id ESTIMATE_ITEM_ID --price 120
./freeagent estimates items delete --id ESTIMATE_ITEM_ID --yes
```

Other write commands (full binary only):

```bash
./freeagent account-locks set --body ./account-lock.json
./freeagent account-locks delete --yes
./freeagent attachments delete --id ATTACHMENT_ID --yes
./freeagent capital-asset-types create --body ./capital-asset-type.json
./freeagent capital-asset-types update --id TYPE_ID --body ./capital-asset-type-update.json
./freeagent capital-asset-types delete --id TYPE_ID --yes
./freeagent journal-sets create --body ./journal-set.json
./freeagent journal-sets update --id JOURNAL_SET_ID --body ./journal-set-update.json
./freeagent journal-sets delete --id JOURNAL_SET_ID --yes
./freeagent notes create --body ./note.json
./freeagent notes update --id NOTE_ID --body ./note-update.json
./freeagent notes delete --id NOTE_ID --yes
./freeagent payroll payment-transition \
  --year 2026 --payment-date 2026-04-30 --name mark_as_paid
./freeagent price-list-items create --body ./price-list-item.json
./freeagent price-list-items update --id ITEM_ID --body ./price-list-item-update.json
./freeagent price-list-items delete --id ITEM_ID --yes
./freeagent properties create --body ./property.json
./freeagent properties update --id PROPERTY_ID --body ./property-update.json
./freeagent properties delete --id PROPERTY_ID --yes
./freeagent sales-tax-periods create --body ./sales-tax-period.json
./freeagent sales-tax-periods update --id PERIOD_ID --body ./sales-tax-period-update.json
./freeagent sales-tax-periods delete --id PERIOD_ID --yes
```

Tax and final accounts:

```bash
./freeagent vat-returns list
./freeagent vat-returns get --period-ends-on 2026-03-31
./freeagent vat-returns transition --period-ends-on 2026-03-31 --name mark_as_filed
./freeagent vat-returns payment-transition \
  --period-ends-on 2026-03-31 --payment-date 2026-04-07 --name mark_as_paid

./freeagent corporation-tax-returns list
./freeagent corporation-tax-returns get --period-ends-on 2026-03-31
./freeagent corporation-tax-returns transition \
  --period-ends-on 2026-03-31 --name mark_as_filed

./freeagent self-assessment-returns list --user USER_ID
./freeagent self-assessment-returns get --user USER_ID --period-ends-on 2026-04-05
./freeagent self-assessment-returns transition \
  --user USER_ID --period-ends-on 2026-04-05 --name mark_as_filed
./freeagent self-assessment-returns payment-transition \
  --user USER_ID --period-ends-on 2026-04-05 --payment-date 2026-07-31 --name mark_as_paid

./freeagent final-accounts-reports list
./freeagent final-accounts-reports get --period-ends-on 2026-03-31
./freeagent final-accounts-reports transition \
  --period-ends-on 2026-03-31 --name mark_as_filed

./freeagent cis-bands list
./freeagent sales-tax-periods list
./freeagent sales-tax-periods get --id PERIOD_ID
```

Accounting transactions (ledger entries — distinct from bank transactions):

```bash
./freeagent transactions list --from-date 2026-01-01 --to-date 2026-03-31
./freeagent transactions list --nominal-code 750-1
./freeagent transactions get --id TRANSACTION_ID
```

Bank transactions (bulk approve):

```bash
./freeagent bank approve \
  --bank-account BANK_ACCOUNT_ID \
  --from 2025-01-01 \
  --to 2025-01-31

./freeagent bank approve --ids ./transaction-ids.txt
./freeagent bank approve --ids ./explanation-ids.txt --ids-type explanation
```

Version and update check:

```bash
./freeagent version              # print version + os/arch, no network call
./freeagent version --check      # compare to latest GitHub release (24h cache)
./freeagent version --check --force  # bypass cache
```

`--check` skips the network entirely on dev / non-tagged builds.

## Files

- Config: `~/.config/freeagent/config.json`
- Tokens (fallback): `~/.config/freeagent/tokens/PROFILE.json`
- Update-check cache: `$XDG_CONFIG_HOME/freeagent/update-check.json`

## Notes

- Default API base URL is production; use `--sandbox` for the sandbox API.
- Use `--json` to print raw JSON for automation or piping into other tools.

## End-to-end tests

This repo ships an opt-in end-to-end harness that drives the CLI's HTTP
client against a real FreeAgent sandbox account. It is gated behind the
`e2e` build tag, skips automatically when its env vars are unset, and is
invoked via:

```bash
make test-e2e
make test-e2e-ro
```

See [`docs/e2e.md`](docs/e2e.md) for sandbox provisioning and env-var
setup.

## Releases

Tagged releases publish cross-platform binaries via GoReleaser and GitHub
Actions. Tags must be cut from a commit on `main`; the release workflow
refuses to run otherwise.

```bash
git checkout main
git pull --ff-only
git tag v0.1.0
git push origin v0.1.0
```

The `release.yml` workflow then runs `goreleaser release --clean`, builds
both `freeagent` and `freeagent-ro` for `darwin`, `linux`, and `windows` on
`amd64` and `arm64`, and publishes the archives plus a `checksums.txt` to
[GitHub Releases](https://github.com/boffinate/freeagent-cli/releases).

For a local dry run without publishing, use `make snapshot` (writes to
`dist/`).

Maintainer notes — including the manual GoReleaser binary bump procedure
that Dependabot can't automate — live in [`RELEASING.md`](./RELEASING.md).

## License

MIT. See `LICENSE`.
