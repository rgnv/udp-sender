package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestNewUDPSender(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("NewUDPSender() error = %v", err)
	}
	if sender == nil {
		t.Fatal("NewUDPSender() returned nil sender without error")
	}
	defer func() { _ = sender.Close() }()

	// Verify both socket file descriptors are valid
	if sender.fdIPv4 < 0 {
		t.Error("IPv4 socket file descriptor is invalid")
	}
	if sender.fdIPv6 < 0 {
		t.Error("IPv6 socket file descriptor is invalid")
	}
}

func TestUDPSender_Send(t *testing.T) {
	requireRoot(t)

	// Start a UDP server to receive messages
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Get the actual port assigned
	serverAddr := conn.LocalAddr().(*net.UDPAddr)

	// Create sender (no arguments needed)
	sender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	tests := []struct {
		name    string
		message string
		srcPort uint16
	}{
		{
			name:    "simple message",
			message: "Hello, UDP!",
			srcPort: 54321,
		},
		{
			name:    "empty message",
			message: "",
			srcPort: 54322,
		},
		{
			name:    "long message",
			message: "This is a longer message to test UDP sending capabilities with raw sockets",
			srcPort: 54323,
		},
	}

	// Parse loopback IP once
	loopbackIP := net.ParseIP("127.0.0.1")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send message with source and destination addresses
			n, err := sender.Send(tt.message, loopbackIP, tt.srcPort, loopbackIP, uint16(serverAddr.Port))
			if err != nil {
				t.Errorf("Send() error = %v", err)
				return
			}
			if n != len(tt.message) {
				t.Errorf("Send() sent %d bytes, want %d", n, len(tt.message))
			}

			// Receive message
			buf := make([]byte, 1024)
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			receivedN, fromAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				t.Errorf("Failed to receive message: %v", err)
				return
			}

			received := string(buf[:receivedN])
			if received != tt.message {
				t.Errorf("Received message = %q, want %q", received, tt.message)
			}

			// Verify source port was spoofed
			if fromAddr.Port != int(tt.srcPort) {
				t.Errorf("Source port = %d, want %d", fromAddr.Port, tt.srcPort)
			}
		})
	}
}

func TestUDPSender_Close(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	err = sender.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Closing again should not panic
	_ = sender.Close()
}

func TestPacketSender_Interface(t *testing.T) {
	requireRoot(t)

	// Test that UDPSender implements PacketSender interface
	var sender PacketSender
	udpSender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = udpSender.Close() }()

	// Assign to interface
	sender = udpSender

	// Test Send through interface with destination
	srcIP := net.ParseIP("10.0.0.1")
	destIP := net.ParseIP("127.0.0.1")
	n, err := sender.Send("test", srcIP, 12345, destIP, 8080)
	if err != nil {
		t.Errorf("Interface Send() error = %v", err)
	}
	if n != 4 {
		t.Errorf("Interface Send() sent %d bytes, want 4", n)
	}
}

func TestRawSocketPermissions(t *testing.T) {
	// This test checks if we have permissions to create raw sockets
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	if err != nil {
		if os.Geteuid() != 0 {
			t.Skip("Raw socket creation requires root privileges (run with sudo)")
		} else {
			t.Errorf("Failed to create raw socket even with root: %v", err)
		}
		return
	}

	_ = syscall.Close(fd)
	t.Log("Successfully created raw socket - running with appropriate privileges")
}

// Benchmarks

func BenchmarkUDPSender_Send(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	// Start a UDP server to receive messages
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		b.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Get the actual port assigned
	serverAddr := conn.LocalAddr().(*net.UDPAddr)

	// Create sender
	sender, err := NewUDPSender()
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Start goroutine to drain received packets
	done := make(chan bool)
	go func() {
		buf := make([]byte, 2048)
		for {
			select {
			case <-done:
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
				_, _, _ = conn.ReadFromUDP(buf)
			}
		}
	}()
	defer func() { done <- true }()

	message := "Benchmark message payload for UDP sender testing"
	loopbackIP := net.ParseIP("127.0.0.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srcPort := uint16(10000 + (i % 55000))
		_, err := sender.Send(message, loopbackIP, srcPort, loopbackIP, uint16(serverAddr.Port))
		if err != nil {
			b.Errorf("Send() error = %v", err)
		}
	}
}

func BenchmarkUDPSender_SendVariablePayloadSizes(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	// Start a UDP server to receive messages
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		b.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer func() { _ = conn.Close() }()

	serverAddr := conn.LocalAddr().(*net.UDPAddr)

	sender, err := NewUDPSender()
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Start goroutine to drain received packets
	done := make(chan bool)
	go func() {
		buf := make([]byte, 65536)
		for {
			select {
			case <-done:
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
				_, _, _ = conn.ReadFromUDP(buf)
			}
		}
	}()
	defer func() { done <- true }()

	loopbackIP := net.ParseIP("127.0.0.1")
	sizes := []int{64, 256, 512, 1024, 4096, 8192}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			payload := string(make([]byte, size))
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				srcPort := uint16(10000 + (i % 55000))
				_, err := sender.Send(payload, loopbackIP, srcPort, loopbackIP, uint16(serverAddr.Port))
				if err != nil {
					b.Errorf("Send() error = %v", err)
				}
			}
		})
	}
}
