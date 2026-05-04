# Usage

Reference for every command the CLI exposes, plus common scenarios that span
multiple commands. Examples use `./freeagent`; substitute `./freeagent-ro` for
read-only invocations.

For exhaustive flag lists, run `freeagent <command> --help`.

## Conventions

- `--json` prints raw JSON on every command that produces output. Use it for
  scripting and piping; the default human format is for terminals only.
- `--sandbox` targets `api.sandbox.freeagent.com` instead of production.
- IDs and URLs are interchangeable for `get`-style commands: pass either
  `--id 12345` or `--url https://api.freeagent.com/v2/invoices/12345`.

## Pagination

Every `list` command auto-paginates by default: the CLI walks the API's
`Link` header (`rel="next"`) and merges all pages into one response. You
get the full result set for the filters you passed, no manual paging.

To bound runaway fetches, auto-pagination stops at 50 pages by default and
emits a warning to stderr asking you to narrow filters or raise `--max-pages`.

Per-list flags (added to every `list` command):

- `--per-page N` — items requested per page (default 100, the API maximum
  for most endpoints; some endpoints like `ap clients list --minimal` allow
  higher values).
- `--page N` — fetch a single page at position N. Disables auto-pagination.
- `--max-pages N` — cap on the auto-pagination walk (default 50).
- `--no-paginate` — disable auto-pagination; return only the first page.

## Invoices

```bash
./freeagent invoices list
./freeagent invoices list --view recent --from 2026-01-01 --to 2026-03-31
./freeagent invoices get --id INVOICE_ID
```

Create a draft invoice. `--contact` accepts a contact ID, URL, name, or email
(name/email are resolved locally against your contacts):

```bash
./freeagent invoices create \
  --contact CONTACT_ID \
  --reference INV-001 \
  --lines ./invoice-lines.json

./freeagent invoices create \
  --contact "Acme Ltd" \
  --reference INV-002 \
  --lines ./invoice-lines.json
```

Send an invoice by email, or mark it as sent without emailing:

```bash
./freeagent invoices send --id INVOICE_ID --email-to you@company.com
./freeagent invoices send --id INVOICE_ID
```

## Contacts

```bash
./freeagent contacts list
./freeagent contacts search --query "Acme"
./freeagent contacts get --id CONTACT_ID
./freeagent contacts create \
  --organisation "Acme Ltd" \
  --email accounts@acme.test
```

## Bank

Accounts, transactions, and explanations (read):

```bash
./freeagent bank accounts list
./freeagent bank accounts get --id BANK_ACCOUNT_ID

./freeagent bank transactions list \
  --bank-account BANK_ACCOUNT_ID \
  --from 2026-01-01 --to 2026-01-31

./freeagent bank explanations list --bank-account BANK_ACCOUNT_ID
./freeagent bank explanations get --id EXPLANATION_ID
```

Bulk-approve transactions or explanations. Either give a date range, or pipe a
file of IDs (one per line):

```bash
./freeagent bank approve \
  --bank-account BANK_ACCOUNT_ID \
  --from 2025-01-01 --to 2025-01-31

./freeagent bank approve --ids ./transaction-ids.txt
./freeagent bank approve --ids ./explanation-ids.txt --ids-type explanation
```

## Bills, expenses, credit notes (read)

```bash
./freeagent bills list --from 2026-01-01 --to 2026-03-31
./freeagent bills get --id BILL_ID

./freeagent expenses list --user USER_ID --from 2026-01-01

./freeagent credit-notes list --contact CONTACT_ID
```

## Projects, tasks, timeslips, estimates

```bash
./freeagent projects list --view active
./freeagent projects get --id PROJECT_ID

./freeagent tasks list --project PROJECT_ID

./freeagent timeslips list \
  --from 2026-01-01 --to 2026-01-31 \
  --user USER_ID

./freeagent estimates list --contact CONTACT_ID
```

## Reports

Reports return raw JSON in both table and `--json` modes:

```bash
./freeagent reports balance-sheet --as-at 2026-03-31
./freeagent reports profit-and-loss --from 2026-01-01 --to 2026-03-31
./freeagent reports trial-balance --from 2026-01-01 --to 2026-03-31
./freeagent reports cashflow --from 2026-01-01 --to 2026-03-31
```

## Reference data

Company, users, categories, price list, and stock items rarely change but are
needed to interpret other endpoints' responses:

```bash
./freeagent company show

./freeagent users list
./freeagent users me

./freeagent categories list
./freeagent price-list-items list
./freeagent stock-items list
```

## raw — break-glass

Use `raw` for any endpoint not wrapped above, or when you need a flag the
wrapper doesn't expose. Only the full `freeagent` binary registers `raw` —
it is not present in `freeagent-ro` at all.

```bash
./freeagent raw --method GET --path /v2/invoices
./freeagent raw --method GET --path "/v2/invoices?view=recent&per_page=100"
```

## Common scenarios

### Period-end pull

Grab the four headline reports as JSON for a given quarter:

```bash
QUARTER_FROM=2026-01-01
QUARTER_TO=2026-03-31

./freeagent reports balance-sheet --as-at $QUARTER_TO --json > bs.json
./freeagent reports profit-and-loss --from $QUARTER_FROM --to $QUARTER_TO --json > pl.json
./freeagent reports trial-balance --from $QUARTER_FROM --to $QUARTER_TO --json > tb.json
./freeagent reports cashflow --from $QUARTER_FROM --to $QUARTER_TO --json > cf.json
```

### Approve a month of bank transactions in one go

```bash
./freeagent bank approve \
  --bank-account BANK_ACCOUNT_ID \
  --from 2026-01-01 --to 2026-01-31
```

### Create and send an invoice in two steps

```bash
INVOICE_ID=$(./freeagent invoices create \
  --contact "Acme Ltd" \
  --reference INV-001 \
  --lines ./invoice-lines.json \
  --json | jq -r '.invoice.id // .id')

./freeagent invoices send --id "$INVOICE_ID" --email-to you@company.com
```
