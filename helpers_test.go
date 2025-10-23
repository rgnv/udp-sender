package main

import (
	"os"
	"testing"
)

// requireRoot skips the test if not running as root/admin or if running in short mode
func requireRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		t.Skip("This test requires root privileges (run with sudo)")
	}
}
