# End-to-end tests

The `internal/e2e` package drives the CLI's HTTP client against a live
FreeAgent **sandbox** account. It is gated behind the `e2e` Go build tag
so it never runs as part of the default `go test ./...`, and every test
will `t.Skip` if the four `FREEAGENT_E2E_*` environment variables are
absent. You can therefore run `make test-e2e` on a fresh checkout and
expect a clean PASS-with-SKIP — no network calls happen unless the
harness is explicitly configured.

This page describes how to wire the harness up against your own sandbox
tenant. **You need a sandbox account; do not point this at production.**

## 1. Create a sandbox account and OAuth app

1. Sign up for a sandbox tenant at <https://signup.sandbox.freeagent.com/signup>.
2. In the sandbox UI, register an OAuth application. Capture the
   `CLIENT_ID` and `CLIENT_SECRET`.
3. Use any valid local redirect URI, e.g. `http://127.0.0.1:8797/callback`.

## 2. Log in via the CLI in sandbox mode

```bash
./bin/freeagent --sandbox auth configure \
  --client-id YOUR_SANDBOX_CLIENT_ID \
  --client-secret YOUR_SANDBOX_CLIENT_SECRET \
  --redirect http://127.0.0.1:8797/callback

./bin/freeagent --sandbox auth login
```

This writes tokens to `~/.config/freeagent/tokens/default.json` (or the
keychain, depending on your build).

## 3. Stage the token file for the harness

The harness reads a single JSON file pointed to by
`FREEAGENT_E2E_TOKEN_FILE`, never the keychain or the user's regular
config dir. Copy the sandbox token into the repo's gitignored `.e2e/`
directory:

```bash
mkdir -p .e2e
cp ~/.config/freeagent/tokens/default.json .e2e/token.json
chmod 600 .e2e/token.json
```

`/.e2e/` is in `.gitignore`, so the file cannot be committed.

## 4. Export the four environment variables

```bash
export FREEAGENT_E2E_TOKEN_FILE="$PWD/.e2e/token.json"
export FREEAGENT_E2E_CLIENT_ID="YOUR_SANDBOX_CLIENT_ID"
export FREEAGENT_E2E_CLIENT_SECRET="YOUR_SANDBOX_CLIENT_SECRET"
# Optional; defaults to the sandbox API root.
export FREEAGENT_E2E_BASE_URL="https://api.sandbox.freeagent.com/v2"
```

If the access token is within five minutes of expiring, the harness
refreshes it on startup and writes the new token back to the same file.

## 5. Run the suites

```bash
make test-e2e        # full build, hits the API where tests do, sweeps e2e- resources
make test-e2e-ro     # readonly build, exercises the read-only call surface
```

Both targets force `-count=1 -parallel 1 -timeout 15m`. Resource creation
helpers stamp every record with an `e2e-<unix-ts>-<uuid>-<test>` prefix;
on entry and exit the suite sweeps stale `e2e-`-prefixed resources so a
crashed previous run doesn't leave litter.

## What ships in this PR vs. follow-ups

This PR lays the harness only:

- env-var contract, token-file auth bootstrap, proactive refresh,
- fixture-prefix helper,
- contacts sweep,
- Makefile targets,
- smoke tests that prove the build wires cleanly without sandbox creds.

Per-command-group coverage (contacts CRUD, invoices CRUD, all read-only
smoke calls, the readonly tripwire suite, and sweeps for the other
resource types) lands in subsequent PRs against issue #8 once a sandbox
account is provisioned.
