# Release Process

The project uses GitHub Actions for automated releases. Simply push a new tag:

```bash
# Ensure you're on the master branch with latest changes
git checkout master
git pull origin master

# Create a new tag (use semantic versioning)
git tag v1.0.3

# Push the tag to GitHub
git push origin v1.0.3
```

The GitHub Actions workflow (`.github/workflows/release.yml`) will automatically:

1. **Run tests** — gates the release; all tests must pass
2. **Build binaries** for:
   - Linux x64
   - Linux ARM64
   - macOS x64 (Intel)
   - macOS ARM64 (Apple Silicon)
3. **Create archives** — each binary compressed into a `.tar.gz` file
4. **Build Linux packages** — DEB (amd64, arm64) and RPM (x86_64, aarch64)
5. **Generate checksums** — SHA256 checksums for all artifacts
6. **Build Docker images** — multi-arch (`linux/amd64`, `linux/arm64`) pushed to `ghcr.io/criblio/udp-sender`
7. **Create GitHub Release** — uploads all artifacts with auto-generated release notes
8. **Update Homebrew formula** — automatically updates `Formula/udp-sender.rb` with version and SHA256 hashes

## Workflow Structure

The release pipeline runs as 6 parallelizable jobs:

| Job | Depends On | Purpose |
|---|---|---|
| `test` | — | Gate: runs `go test -v ./...`, extracts version, detects pre-release |
| `build` | `test` | Cross-compiles binaries, creates archives, builds DEB/RPM packages |
| `docker` | `test` | Builds and pushes multi-arch container images (parallel with `build`) |
| `release` | `build`, `docker` | Generates changelog, creates GitHub Release with all artifacts |
| `homebrew` | `release` | Downloads macOS binaries, computes SHAs, updates `Formula/udp-sender.rb`, commits |
| `summary` | `release`, `homebrew` | Prints release summary to GitHub Actions UI |

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: Backwards-compatible functionality additions
- **PATCH** version: Backwards-compatible bug fixes

Examples:

- `v1.0.0` — Initial release
- `v1.1.0` — New feature added
- `v1.1.1` — Bug fix
- `v2.0.0` — Breaking change

## Pre-release Versions

For beta, release candidate, or pre-release versions, use a tag containing a hyphen:

```bash
git tag v1.0.3-beta.1
git tag v1.0.3-rc.1
git tag v1.1.0-pre
```

The workflow automatically detects pre-releases (any tag containing `-`) and marks the GitHub Release accordingly. Pre-releases are **not** tagged as `latest` in Docker.

## Homebrew

macOS users can install via Homebrew:

```bash
brew tap criblio/udp-sender
brew install udp-sender
```

The formula (`Formula/udp-sender.rb`) is automatically updated by the release workflow with the correct version and SHA256 hashes for both Intel and Apple Silicon Macs. No manual intervention needed.

To test the formula locally before a release:

```bash
brew install --build-from-source ./Formula/udp-sender.rb
```

## Release Checklist

Before creating a release:

- [ ] All tests pass: `sudo make test-root`
- [ ] Code is linted: `make lint`
- [ ] Documentation is updated (README.md, PROTOCOL.md, RELEASING.md)
- [ ] Version number follows semantic versioning
- [ ] All changes are committed and pushed to main

## Verifying a Release

After the release is created:

1. **Check GitHub Actions**:
   - Visit: https://github.com/criblio/udp-sender/actions
   - Verify all 6 jobs in the "Release" workflow completed successfully

2. **Check GitHub Releases**:
   - Visit: https://github.com/criblio/udp-sender/releases
   - Verify all binaries, packages, and checksums are present

3. **Test a binary**:

   ```bash
   # Download
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-linux-x64.tar.gz

   # Extract and test
   tar -xzf udp-sender-v1.0.3-linux-x64.tar.gz
   ./udp-sender-linux-x64 -h
   ```

4. **Test Homebrew install**:

   ```bash
   brew update
   brew install udp-sender
   udp-sender --version
   ```

5. **Verify checksums**:

   Each release artifact includes a corresponding `.sha256` file for verification.

   **For standalone binaries:**

   ```bash
   # Download the archive and its checksum
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-linux-x64.tar.gz
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-linux-x64.tar.gz.sha256

   # Verify the checksum
   sha256sum -c udp-sender-v1.0.3-linux-x64.tar.gz.sha256
   ```

   Expected output:

   ```text
   udp-sender-v1.0.3-linux-x64.tar.gz: OK
   ```

   **For Linux packages (DEB/RPM):**

   ```bash
   # Download the package and its checksum
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-1.0.3-x64.deb
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-1.0.3-x64.deb.sha256

   # Verify the checksum
   sha256sum -c udp-sender-1.0.3-x64.deb.sha256
   ```

   **On macOS**, use `shasum` instead:

   ```bash
   # Download macOS archive and checksum
   curl -LO https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-darwin-arm64.tar.gz
   curl -LO https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-darwin-arm64.tar.gz.sha256

   # Verify the checksum
   shasum -a 256 -c udp-sender-v1.0.3-darwin-arm64.tar.gz.sha256
   ```

   **Manual verification** (if you prefer to compare checksums yourself):

   ```bash
   # Calculate checksum
   sha256sum udp-sender-v1.0.3-linux-x64.tar.gz

   # Display expected checksum
   cat udp-sender-v1.0.3-linux-x64.tar.gz.sha256

   # The two checksums should match exactly
   ```

## Troubleshooting

### Release workflow fails

- Check GitHub Actions logs for specific errors
- The workflow runs as 6 jobs — identify which job failed
- Ensure `GITHUB_TOKEN` has appropriate permissions (`contents: write`, `packages: write`)
- Verify `go.mod` and dependencies are correct

### Homebrew formula not updated

- Check the `homebrew` job in the release workflow
- The formula is committed and pushed back to the repo — ensure the `github-actions[bot]` has push permissions
- Verify `Formula/udp-sender.rb` exists and has valid placeholders

### Binary doesn't work on target platform

- Verify correct `GOOS` and `GOARCH` in workflow
- Check for platform-specific syscall issues
- Test cross-compilation locally before pushing tag

### Checksums don't match

- Re-download the binary (may be corruption)
- Check if binary was modified after release
- Verify `checksums.txt` wasn't modified

## Rolling Back a Release

If you need to remove a release:

1. **Delete the release on GitHub**:
   - Go to Releases page
   - Click the release
   - Click "Delete release"

2. **Delete the Docker tag** (optional):
   ```bash
   gh api --method DELETE "/orgs/criblio/packages/container/udp-sender/versions/VERSION_ID"
   ```

3. **Delete the tag**:

   ```bash
   # Delete local tag
   git tag -d v1.0.3

   # Delete remote tag
   git push origin :refs/tags/v1.0.3
   ```

4. **Create a new fixed release** with the next version number
