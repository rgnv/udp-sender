# GitHub Actions Workflows

This directory contains GitHub Actions workflows for CI/CD.

## Workflows

### CI Workflow (`ci.yml`)

**Trigger**: Push to `master`/`main` branch, Pull Requests

**Purpose**: Continuous Integration

**Jobs**:

| Job | Purpose |
|---|---|
| `test-unit` | Unit tests on Go 1.24 and 1.25 (short mode, `-race`) |
| `test-integration` | Integration tests with `NET_RAW` + `NET_ADMIN` capabilities in container |
| `build` | Verifies the project builds successfully |
| `lint` | Runs golangci-lint for code quality |
| `benchmark` | Runs benchmarks and tracks performance history |

**Status Badge**:

```markdown
[![CI](https://github.com/criblio/udp-sender/actions/workflows/ci.yml/badge.svg)](https://github.com/criblio/udp-sender/actions/workflows/ci.yml)
```

### Release Workflow (`release.yml`)

**Trigger**: Push tags matching `v*.*.*` (e.g., `v1.0.3`)

**Purpose**: Automated release pipeline with 6 parallelizable jobs

**Jobs**:

| Job | Depends On | Purpose |
|---|---|---|
| `test` | — | Gate: `go test -v ./...` must pass |
| `build` | `test` | Cross-compiles 4 binaries, creates `.tar.gz` archives + SHA256 checksums, builds DEB/RPM packages via `fpm` |
| `docker` | `test` | Builds and pushes multi-arch Docker images (`linux/amd64`, `linux/arm64`) to `ghcr.io` |
| `release` | `build`, `docker` | Generates changelog from conventional commits, creates GitHub Release with all artifacts |
| `homebrew` | `release` | Downloads macOS binaries, computes SHA256, updates `Formula/udp-sender.rb`, commits back to repo |
| `summary` | `release`, `homebrew` | Prints release summary to GitHub Actions job summary |

**Pre-release detection**: Tags containing `-` (e.g., `v1.0.3-beta.1`) are automatically marked as pre-releases.

**Outputs**:

| Type | Files |
|---|---|
| Standalone archives | `udp-sender-{VERSION}-{linux\|darwin}-{x64\|arm64}.tar.gz` + `.sha256` |
| DEB packages | `udp-sender-{PKG_VERSION}-{x64\|arm64}.deb` + `.sha256` |
| RPM packages | `udp-sender-{PKG_VERSION}-{x64\|arm64}.rpm` + `.sha256` |
| Checksums | Individual `.sha256` files + combined `checksums.txt` |
| Docker images | `ghcr.io/criblio/udp-sender:{version}`, `latest`, `{major}.{minor}`, `{major}` |
| Homebrew | `Formula/udp-sender.rb` (auto-updated with new version + SHA256) |

## Creating a Release

1. Ensure all changes are committed and pushed to `master`
2. Create and push a tag:

   ```bash
   # Stable release
   git tag v1.0.3
   git push origin v1.0.3

   # Pre-release
   git tag v1.0.3-beta.1
   git push origin v1.0.3-beta.1
   ```

3. GitHub Actions automatically builds and releases
4. Check the release at: https://github.com/criblio/udp-sender/releases

## Permissions

| Workflow | Permissions |
|---|---|
| CI | `contents: read` (default) |
| Release — `test`, `build`, `docker`, `release` | `contents: write`, `packages: write` |
| Release — `homebrew` | `contents: write` (to commit formula update) |

## Docker Images

Container images are published to GitHub Container Registry (`ghcr.io`) with each release:

```bash
# Pull and run the latest version
docker pull ghcr.io/criblio/udp-sender:latest
cat packets.bin | docker run --rm -i --cap-add=NET_RAW ghcr.io/criblio/udp-sender:latest

# Pull a specific version
docker pull ghcr.io/criblio/udp-sender:1.0.3
```

**Tags:**

- `latest` — Most recent stable release
- `X.Y.Z` — Specific version (e.g., `1.0.3`)
- `X.Y` — Minor version (e.g., `1.0`)
- `X` — Major version (e.g., `1`)

**Architectures:** `linux/amd64`, `linux/arm64`

View images at: https://github.com/criblio/udp-sender/pkgs/container/udp-sender

## Homebrew

macOS users can install via Homebrew using the formula in this repo:

```bash
brew tap criblio/udp-sender
brew install udp-sender
```

The formula (`Formula/udp-sender.rb`) supports both Intel and Apple Silicon Macs, with SHA256 verification for each architecture. It is automatically updated by the release workflow — no manual maintenance required.

## Monitoring

View workflow runs at: https://github.com/criblio/udp-sender/actions
