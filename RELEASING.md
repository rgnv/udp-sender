# Release Process

This document describes how to create a new release of UDP Sender.

## Automated Release (Recommended)

The project uses GitHub Actions for automated releases. Simply push a new tag:

```bash
# Ensure you're on the master branch with latest changes
git checkout master
git pull origin master

# Create a new tag (use semantic versioning)
git tag v1.0.0

# Push the tag to GitHub
git push origin v1.0.0
```

The GitHub Actions workflow (`.github/workflows/release.yml`) will automatically:

1. **Build binaries** for:
   - Linux x64
   - Linux ARM64
   - macOS x64 (Intel)
   - macOS ARM64 (Apple Silicon)

2. **Create archives**:
   - Each binary is compressed into a `.tar.gz` file
   - Archives include the binary and documentation

3. **Generate checksums**:
   - SHA256 checksums for all binaries and archives
   - Saved in `checksums.txt`

4. **Create GitHub Release**:
   - Automatically creates a GitHub release
   - Includes release notes with download instructions
   - Uploads all binaries, archives, and checksums

5. **Publish**:
   - Release is immediately available
   - Badges and links are automatically updated

## Manual Release (Alternative)

If you prefer to use GoReleaser locally:

### Prerequisites

```bash
# Install GoReleaser
brew install goreleaser

# Or using Go
go install github.com/goreleaser/goreleaser@latest
```

### Steps

1. **Set up GitHub token**:

   ```bash
   export GITHUB_TOKEN="your_github_personal_access_token"
   ```

2. **Create and push tag**:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. **Run GoReleaser**:

   ```bash
   goreleaser release --clean
   ```

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: Backwards-compatible functionality additions
- **PATCH** version: Backwards-compatible bug fixes

Examples:

- `v1.0.0` - Initial release
- `v1.1.0` - New feature added
- `v1.1.1` - Bug fix
- `v2.0.0` - Breaking change

## Pre-release Versions

For beta or release candidate versions:

```bash
git tag v1.0.0-beta.1
git tag v1.0.0-rc.1
```

These will be marked as "pre-release" on GitHub.

## Release Checklist

Before creating a release:

- [ ] All tests pass: `go test ./...`
- [ ] Code is linted: `golangci-lint run`
- [ ] Documentation is updated (README.md, PROTOCOL.md)
- [ ] CHANGELOG.md is updated (if applicable)
- [ ] Version number follows semantic versioning
- [ ] All changes are committed and pushed to master

## Verifying a Release

After the release is created:

1. **Check GitHub Actions**:
   - Visit: <https://github.com/criblio/udp-sender/actions>
   - Verify the "Release" workflow completed successfully

2. **Check GitHub Releases**:
   - Visit: <https://github.com/criblio/udp-sender/releases>
   - Verify all binaries and checksums are present

3. **Test a binary**:

   ```bash
   # Download
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.0/udp-sender-v1.0.0-linux-x64.tar.gz
   
   # Extract and test
   tar -xzf udp-sender-v1.0.0-linux-x64.tar.gz
   ./udp-sender-linux-x64 -h
   ```

4. **Verify checksums**:

   ```bash
   wget https://github.com/criblio/udp-sender/releases/download/v1.0.0/checksums.txt
   sha256sum -c checksums.txt --ignore-missing
   ```

## Troubleshooting

### Release workflow fails

- Check GitHub Actions logs for specific errors
- Ensure GITHUB_TOKEN has appropriate permissions
- Verify go.mod and dependencies are correct

### Binary doesn't work on target platform

- Verify correct GOOS and GOARCH in workflow
- Check for platform-specific syscall issues
- Test cross-compilation locally before pushing tag

### Checksums don't match

- Re-download the binary (may be corruption)
- Check if binary was modified after release
- Verify checksums.txt wasn't modified

## Rolling Back a Release

If you need to remove a release:

1. **Delete the release on GitHub**:
   - Go to Releases page
   - Click the release
   - Click "Delete release"

2. **Delete the tag**:

   ```bash
   # Delete local tag
   git tag -d v1.0.0
   
   # Delete remote tag
   git push origin :refs/tags/v1.0.0
   ```

3. **Create a new fixed release** with the next version number
