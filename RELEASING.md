# Releasing

This document is for maintainers cutting a release. End-user install
instructions live in `README.md`.

## Cutting a release

```bash
git checkout main
git pull --ff-only
git tag v0.1.0
git push origin v0.1.0
```

The `release.yml` workflow runs `goreleaser release --clean` against the
tag, builds both binaries for darwin/linux/windows on amd64/arm64, and
publishes the archives plus `checksums.txt` to GitHub Releases. The
workflow refuses to run if the tag commit is not reachable from
`origin/main`.

For a local dry run without publishing, use `make snapshot` (writes to
`dist/`).

## Manual maintenance: GoReleaser binary version

The release workflow pins the **GoReleaser binary** to an exact version
because the binary runs inside a job with `contents: write` and produces
the artifacts users download. Dependabot **cannot** manage this pin: the
field is `with: version:` on `goreleaser/goreleaser-action`, which is an
action input rather than a versioned action reference, so it is invisible
to the `github-actions` ecosystem updater.

This means the pin will go stale unless a maintainer bumps it. Plan to
review it at least once per quarter, or when cutting any release after a
long gap.

### Bump procedure

1. Check the current pin in `.github/workflows/release.yml`:
   ```bash
   grep 'version: v' .github/workflows/release.yml
   ```
2. Find the latest stable GoReleaser release (skip prereleases /
   nightlies):
   ```bash
   gh release list --repo goreleaser/goreleaser --limit 10
   ```
3. Read the changelog between the current pin and the candidate version,
   especially looking for breaking changes to the config schema or to
   the build/archive/release stages we use:
   <https://github.com/goreleaser/goreleaser/releases>
4. Update `.github/workflows/release.yml`:
   ```yaml
   version: v2.X.Y   # exact, no leading "~>"
   ```
5. Validate locally before opening the PR:
   ```bash
   goreleaser check
   make snapshot     # full dry-run end-to-end
   ```
6. Open a PR titled `ci(release): bump GoReleaser to v2.X.Y` with the
   relevant changelog highlights in the body. Merge on green.

### What automation already covers

- **Action SHAs** (`actions/checkout`, `actions/setup-go`,
  `goreleaser/goreleaser-action`): managed by `.github/dependabot.yml`
  with a 7-day cooldown.
- **Go module dependencies**: same Dependabot config, weekly schedule,
  7-day cooldown.

Only the GoReleaser binary version needs a human.
