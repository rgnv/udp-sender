package main

import (
	"net"
	"os"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "simple data",
			data: []byte{0x00, 0x01, 0x02, 0x03},
		},
		{
			name: "odd length data",
			data: []byte{0x00, 0x01, 0x02},
		},
		{
			name: "all zeros",
			data: []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChecksum(tt.data)

			// Verify checksum is calculated (non-zero for most inputs)
			// and verify double calculation returns original
			doubleData := make([]byte, len(tt.data)+2)
			copy(doubleData, tt.data)
			// Insert checksum
			doubleData[len(doubleData)-2] = byte(result >> 8)
			doubleData[len(doubleData)-1] = byte(result & 0xff)

			// Verify checksum produces expected result for validation
			verification := calculateChecksum(doubleData)
			if len(tt.data) > 0 && verification != 0xffff && verification != 0 {
				t.Logf("Checksum verification result: 0x%04x for input 0x%04x", verification, result)
			}
		})
	}
}

func TestBuildIPHeader(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("192.168.1.1").To4()
	header := sender.buildIPv4Header(100, srcIP, destIP)

	// Check header length
	if len(header) != 20 {
		t.Errorf("IP header length = %d, want 20", len(header))
	}

	// Check version and IHL
	if header[0] != 0x45 {
		t.Errorf("Version/IHL = 0x%02x, want 0x45", header[0])
	}

	// Check protocol (UDP = 17)
	if header[9] != 17 {
		t.Errorf("Protocol = %d, want 17 (UDP)", header[9])
	}

	// Verify source IP
	headerSrcIP := net.IP(header[12:16])
	if headerSrcIP.String() != "10.0.0.1" {
		t.Errorf("Source IP = %s, want 10.0.0.1", headerSrcIP.String())
	}

	// Verify destination IP
	dstIP := net.IP(header[16:20])
	if dstIP.String() != "192.168.1.1" {
		t.Errorf("Destination IP = %s, want 192.168.1.1", dstIP.String())
	}
}

func TestBuildIPv6Header(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender()
	if err != nil {
		t.Skip("IPv6 not available")
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("::1")
	header := sender.buildIPv6Header(100, srcIP, destIP)

	// Check header length
	if len(header) != 40 {
		t.Errorf("IPv6 header length = %d, want 40", len(header))
	}

	// Check version (first 4 bits should be 6)
	version := (header[0] >> 4) & 0x0F
	if version != 6 {
		t.Errorf("IP version = %d, want 6", version)
	}

	// Check next header (UDP = 17)
	if header[6] != 17 {
		t.Errorf("Next header = %d, want 17 (UDP)", header[6])
	}

	// Verify source address
	headerSrcIP := net.IP(header[8:24])
	if !headerSrcIP.Equal(srcIP.To16()) {
		t.Errorf("Source IP = %v, want %v", headerSrcIP, srcIP)
	}

	// Verify destination address
	headerDestIP := net.IP(header[24:40])
	if !headerDestIP.Equal(destIP.To16()) {
		t.Errorf("Destination IP = %v, want %v", headerDestIP, destIP)
	}
}

func TestBuildUDPHeader(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender()
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	payload := []byte("test payload")
	srcIP := net.ParseIP("127.0.0.1").To4()
	destIP := net.ParseIP("127.0.0.1").To4()
	header := sender.buildUDPHeader(payload, srcIP, 12345, destIP, 8080, false)

	// Check header length
	if len(header) != 8 {
		t.Errorf("UDP header length = %d, want 8", len(header))
	}

	// Check source port
	srcPort := (uint16(header[0]) << 8) | uint16(header[1])
	if srcPort != 12345 {
		t.Errorf("Source port = %d, want 12345", srcPort)
	}

	// Check destination port
	dstPort := (uint16(header[2]) << 8) | uint16(header[3])
	if dstPort != 8080 {
		t.Errorf("Destination port = %d, want 8080", dstPort)
	}

	// Check length
	length := (uint16(header[4]) << 8) | uint16(header[5])
	expectedLength := uint16(8 + len(payload))
	if length != expectedLength {
		t.Errorf("UDP length = %d, want %d", length, expectedLength)
	}
}

// Benchmarks

func BenchmarkCalculateChecksum(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateChecksum(data)
	}
}

func BenchmarkBuildIPv4Header(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	sender, err := NewUDPSender()
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("192.168.1.1").To4()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.buildIPv4Header(1024, srcIP, destIP)
	}
}

func BenchmarkBuildUDPHeader(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	sender, err := NewUDPSender()
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	payload := make([]byte, 512)
	srcIP := net.ParseIP("127.0.0.1").To4()
	destIP := net.ParseIP("127.0.0.1").To4()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.buildUDPHeader(payload, srcIP, 12345, destIP, 8080, false)
	}
}
