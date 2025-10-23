package main

import (
	"flag"
	"fmt"
	"os"
)

// Version is the current version of udp-sender
// This can be overridden at build time using: go build -ldflags "-X main.Version=v1.0.0"
var Version = "dev"

func main() {
	// Define flags
	var showVersion bool
	var verbose bool

	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showVersion, "V", false, "Print version and exit (short)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging (debug level)")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging (debug level, short)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Send UDP packets with IP/port spoofing using raw sockets.\n")
		fmt.Fprintf(os.Stderr, "Requires root/administrator privileges (or the CAP_NET_RAW capability).\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -h, --help       Show this help message\n")
		fmt.Fprintf(os.Stderr, "  -V, --version    Print version and exit\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose    Enable verbose logging (debug level)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Reads packets from stdin using a binary protocol (network byte order/big endian).\n")
		fmt.Fprintf(os.Stderr, "Each packet specifies its own source and destination:\n")
		fmt.Fprintf(os.Stderr, "  - Magic:          3 bytes (0xC1 0x21 0xB1)\n")
		fmt.Fprintf(os.Stderr, "  - Version:        1 byte (4 = IPv4, 6 = IPv6)\n")
		fmt.Fprintf(os.Stderr, "  - Source IP:      4 bytes (IPv4) or 16 bytes (IPv6)\n")
		fmt.Fprintf(os.Stderr, "  - Dest IP:        4 bytes (IPv4) or 16 bytes (IPv6)\n")
		fmt.Fprintf(os.Stderr, "  - Source Port:    2 bytes (uint16)\n")
		fmt.Fprintf(os.Stderr, "  - Dest Port:      2 bytes (uint16)\n")
		fmt.Fprintf(os.Stderr, "  - Payload Length: 2 bytes (uint16)\n")
		fmt.Fprintf(os.Stderr, "  - Payload:        N bytes (up to 65535)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  cat packets.bin | sudo %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  ./packet-generator | sudo %s\n", os.Args[0])
	}

	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Printf("udp-sender version %s\n", Version)
		os.Exit(0)
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
		logger.Fatal("Error creating UDP sender", map[string]any{
			"error": err.Error(),
		})
	}
	defer sender.Close()

	// Process stream from stdin
	err = processInputStream(logger, sender, os.Stdin)
	if err != nil {
		logger.Fatal("Error processing stream", map[string]any{
			"error": err.Error(),
		})
	}
}
