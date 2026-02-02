# freegant

A small CLI for the FreeAgent API, built in Go.

## Features

- OAuth login (local callback or manual paste)
- Keychain-backed token storage with file fallback
- Create and send invoices
- Break-glass `raw` command for any FreeAgent endpoint
- JSON output mode for scripting / agents

## Install

```bash
go build ./cmd/freegant
```

## Configure

Create a FreeAgent API application and note the client ID + secret.

Save app credentials:

```bash
./freegant auth configure \
  --client-id YOUR_ID \
  --client-secret YOUR_SECRET \
  --redirect http://127.0.0.1:8797/callback
```

You can also use env vars:

```bash
export FREEGANT_CLIENT_ID=...
export FREEGANT_CLIENT_SECRET=...
export FREEGANT_REDIRECT_URI=http://127.0.0.1:8797/callback
```

## Login

Local callback (default):

```bash
./freegant auth login
```

Manual flow:

```bash
./freegant auth login --manual
```

## Usage

Create a draft invoice:

```bash
./freegant invoices create \
  --contact CONTACT_ID \
  --reference INV-001 \
  --lines ./invoice-lines.json
```

Send an invoice email:

```bash
./freegant invoices send --id INVOICE_ID --email-to you@company.com
```

Mark as sent (no email):

```bash
./freegant invoices send --id INVOICE_ID
```

Break-glass request:

```bash
./freegant raw --method GET --path /v2/invoices
```

## Files

- Config: `~/.config/freegant/config.json`
- Tokens (fallback): `~/.config/freegant/tokens/PROFILE.json`

## Notes

- Default API base URL is production; use `--sandbox` for the sandbox API.
- Use `--json` to print raw JSON for automation or piping into other tools.

## License

MIT. See `LICENSE`.
