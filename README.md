# freeagent

A small CLI for the FreeAgent API, built in Go.

## Features

- OAuth login (local callback or manual paste)
- Keychain-backed token storage with file fallback
- Create and send invoices
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

1. **Command-tree exclusion** via Go build tags. `freeagent-ro` does not register `bank`, `raw`, `contacts create`, `invoices create`, `invoices send`, or `invoices delete`.
2. **HTTP-client guard**. Under `-tags readonly`, every request must satisfy all of:
   - https scheme (no plaintext HTTP — bearer tokens must not traverse unencrypted),
   - host in `{api.freeagent.com, api.sandbox.freeagent.com}` (blocks bearer-token exfiltration via `--base-url` or server-returned absolute URLs),
   - method is GET/HEAD, **or** POST to the exact path `/v2/token_endpoint` (OAuth flow).

   This catches the case where a future refactor adds a mutating call inside a read subcommand.

## Install

### Full (read + write) binary

```bash
go install github.com/anjor/freeagent-cli/cmd/freeagent@latest
```

Or build from source:

```bash
make build   # produces bin/freeagent
```

### Read-only binary

```bash
git clone https://github.com/anjor/freeagent-cli.git
cd freeagent-cli
make install-ro           # installs to $GOPATH/bin/freeagent-ro
# or, to a custom location:
PREFIX=/usr/local/bin make install-ro
```

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
./freeagent users me
./freeagent categories list
./freeagent price-list-items list
./freeagent stock-items list
```

Projects, tasks, timeslips, estimates:

```bash
./freeagent projects list --view active
./freeagent projects get --id PROJECT_ID
./freeagent tasks list --project PROJECT_ID
./freeagent timeslips list --from 2026-01-01 --to 2026-01-31 --user USER_ID
./freeagent estimates list --contact CONTACT_ID
```

Bills, expenses, credit notes (read):

```bash
./freeagent bills list --from 2026-01-01 --to 2026-03-31
./freeagent bills get --id BILL_ID
./freeagent expenses list --user USER_ID --from 2026-01-01
./freeagent credit-notes list --contact CONTACT_ID
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

## Files

- Config: `~/.config/freeagent/config.json`
- Tokens (fallback): `~/.config/freeagent/tokens/PROFILE.json`

## Notes

- Default API base URL is production; use `--sandbox` for the sandbox API.
- Use `--json` to print raw JSON for automation or piping into other tools.

## License

MIT. See `LICENSE`.
