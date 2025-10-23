# Testing Strategy

This project uses a hybrid testing approach to enable comprehensive testing both locally and in CI without requiring root privileges for all tests.

## Test Organization

### Test Files

- **`helpers_test.go`** - Common test helpers
  - `requireRoot()` - Checks for root privileges and respects `-short` flag
  
- **`packet_test.go`** - Packet construction tests
  - Unit tests: Checksum calculations (no root needed)
  - Integration tests: IP/UDP header building (requires root)
  - Benchmarks: Checksum and header building performance
  
- **`sender_test.go`** - UDPSender tests
  - Unit tests: Input validation, error handling (no root needed)
  - Integration tests: Actual packet sending with spoofing (requires root)
  - Benchmarks: Send performance tests

## Running Tests

### Local Development

```bash
# Run all tests (requires sudo for integration tests)
sudo go test -v ./...

# Run only unit tests (no sudo needed)
go test -short -v ./...

# Run only integration tests (requires sudo)
sudo go test -v -run TestUDPSender_Send ./...

# Run benchmarks (unit tests only, no sudo)
go test -short -bench=. -benchmem -run=^$ ./...

# Run all benchmarks (requires sudo)
sudo go test -bench=. -benchmem -run=^$ ./...
```

### CI Pipeline

The CI automatically runs both unit and integration tests:

#### 1. Unit Tests (`test-unit` job)

- **Runs on:** Ubuntu (standard runner)
- **Command:** `go test -short -v -race ./...`
- **Coverage:** Unit tests only (no root privileges)
- **Matrix:** Go 1.21 and 1.22
- **Fast:** ~10-20 seconds

#### 2. Integration Tests (`test-integration` job)

- **Runs in:** Docker container with `CAP_NET_RAW` capability
- **Command:** `go test -v ./...` (full test suite)
- **Coverage:** All tests including raw socket operations
- **Single version:** Go 1.21
- **Slower:** ~30-60 seconds (Docker overhead)

#### 3. Benchmarks (`benchmark` job)

- **Runs on:** Ubuntu (standard runner)
- **Command:** `go test -short -bench=. -benchmem ./...`
- **Coverage:** Non-root benchmarks only
- **Tracked:** Results stored and tracked over time

## Test Categories

### Unit Tests (No Root Required)

These tests run in short mode (`-short` flag):

- ✅ Checksum calculations
- ✅ Input validation
- ✅ Error handling
- ✅ Port parsing
- ✅ Configuration validation

**Benefits:**

- Fast execution
- No special privileges needed
- Can run anywhere (CI, dev machines, containers)
- Immediate feedback

### Integration Tests (Root Required)

These tests are skipped in short mode:

- 🔒 Raw socket creation
- 🔒 IP header construction
- 🔒 UDP header construction
- 🔒 Actual packet sending
- 🔒 Source IP/port spoofing verification

**Benefits:**

- Test real behavior
- Verify raw socket operations
- End-to-end validation
- Full coverage

## The `-short` Flag

Tests use Go's built-in `-short` flag to determine which tests to run:

```go
func requireRoot(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping test that requires root privileges in short mode")
    }
    if os.Geteuid() != 0 {
        t.Skip("This test requires root privileges (run with sudo)")
    }
}
```

This approach:

- Uses standard Go testing patterns
- Requires no custom build tags
- Works with all standard Go tools
- Clear and explicit test categorization

## Docker Container Capabilities

The integration test job uses Docker with specific Linux capabilities:

```yaml
container:
  image: golang:1.21
  options: --cap-add=NET_RAW --cap-add=NET_ADMIN
```

**Why this works:**

- `CAP_NET_RAW` - Allows raw socket creation (required for IP spoofing)
- `CAP_NET_ADMIN` - Allows network configuration (required for some operations)
- More secure than running as root (principle of least privilege)
- Provides same capabilities as `sudo setcap cap_net_raw+ep`

## Coverage Reporting

Both test jobs report coverage separately:

- **Unit tests:** Flag `unittests` - What works without privileges
- **Integration tests:** Flag `integration` - Full system behavior

Combined coverage shows complete test coverage across all scenarios.

## Best Practices

### When Writing Tests

1. **Default to unit tests** - Don't require root unless necessary
2. **Use `requireRoot()`** - For tests that need raw sockets
3. **Test validation first** - Input checking doesn't need root
4. **Separate concerns** - Unit test logic, integration test behavior

### When Running Locally

1. **Start with short mode** - Fast feedback loop

   ```bash
   go test -short -v ./...
   ```

2. **Run integration tests before committing**

   ```bash
   sudo go test -v ./...
   ```

3. **Check benchmarks periodically**

   ```bash
   go test -short -bench=BenchmarkCalculateChecksum -benchmem
   ```

## Debugging Failed Tests

### Unit Test Failures

- No special setup needed
- Use standard debugging tools
- Fast iteration cycle

### Integration Test Failures  

- Ensure running with sudo/capabilities
- Check network permissions
- Verify firewall settings
- Look for port conflicts

## Future Improvements

Possible enhancements:

1. **Mock socket layer** - Test more logic without raw sockets
2. **Test containers** - Pre-built container images
3. **Parallel execution** - Speed up integration tests
4. **Performance regression detection** - Automated alerts
5. **Cross-platform testing** - Add macOS integration tests
