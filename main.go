package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Version is the current version of udp-sender
// This can be overridden at build time using: go build -ldflags "-X main.Version=v1.0.0"
var Version = "dev"

// run contains the core application logic and is testable
// It returns an error instead of calling os.Exit, making it easier to test
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Create a new flag set for testing purposes
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)

	// Define flags
	var showVersion bool
	var verbose bool

	fs.BoolVar(&showVersion, "version", false, "Print version and exit")
	fs.BoolVar(&showVersion, "V", false, "Print version and exit (short)")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose logging (debug level)")
	fs.BoolVar(&verbose, "v", false, "Enable verbose logging (debug level, short)")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [OPTIONS]\n\n", args[0])
		fmt.Fprintf(stderr, "Send UDP packets with IP/port spoofing using raw sockets.\n")
		fmt.Fprintf(stderr, "Requires root/administrator privileges (or the CAP_NET_RAW capability).\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fmt.Fprintf(stderr, "  -h, --help       Show this help message\n")
		fmt.Fprintf(stderr, "  -V, --version    Print version and exit\n")
		fmt.Fprintf(stderr, "  -v, --verbose    Enable verbose logging (debug level)\n")
		fmt.Fprintf(stderr, "\n")
		fmt.Fprintf(stderr, "Reads packets from stdin using a binary protocol.\n")
		fmt.Fprintf(stderr, "See PROTOCOL.md for complete protocol specification.\n\n")
		fmt.Fprintf(stderr, "Examples:\n")
		fmt.Fprintf(stderr, "  cat packets.bin | sudo %s\n", args[0])
		fmt.Fprintf(stderr, "  ./packet-generator | sudo %s\n", args[0])
	}

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Handle version flag
	if showVersion {
		fmt.Fprintf(stdout, "udp-sender version %s\n", Version)
		return nil
	}

	// Create logger with appropriate level
	var logger *Logger
	if verbose {
		logger = NewLoggerWithLevel(LogLevelDebug)
	} else {
		logger = NewLogger()
	}

	// Create sender (no destination needed - comes from packets)
	sender, err := NewUDPSender()
	if err != nil {
		return fmt.Errorf("error creating UDP sender: %w", err)
	}
	defer func() {
		if err := sender.Close(); err != nil {
			logger.Error("Error closing sender", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	// Process stream from stdin
	err = processInputStream(logger, sender, stdin)
	if err != nil {
		return fmt.Errorf("error processing stream: %w", err)
	}

	return nil
}

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
