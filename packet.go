package main

import (
	"encoding/binary"
	"net"
)

// buildPacket constructs a complete IP + UDP packet for IPv4 or IPv6
func (s *UDPSender) buildPacket(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) []byte {
	isIPv6 := srcIP.To4() == nil

	if isIPv6 {
		ipHeader := s.buildIPv6Header(len(payload)+8, srcIP, destIP) // 8 bytes for UDP header
		udpHeader := s.buildUDPHeader(payload, srcIP, srcPort, destIP, destPort, isIPv6)

		packet := make([]byte, 0, len(ipHeader)+len(udpHeader)+len(payload))
		packet = append(packet, ipHeader...)
		packet = append(packet, udpHeader...)
		packet = append(packet, payload...)

		return packet
	} else {
		ipHeader := s.buildIPv4Header(len(payload)+8, srcIP, destIP) // 8 bytes for UDP header
		udpHeader := s.buildUDPHeader(payload, srcIP, srcPort, destIP, destPort, isIPv6)

		packet := make([]byte, 0, len(ipHeader)+len(udpHeader)+len(payload))
		packet = append(packet, ipHeader...)
		packet = append(packet, udpHeader...)
		packet = append(packet, payload...)

		return packet
	}
}

// buildIPv4Header constructs an IPv4 header
func (s *UDPSender) buildIPv4Header(dataLen int, srcIP net.IP, destIP net.IP) []byte {
	header := make([]byte, 20)

	// Version (4 bits) and IHL (4 bits)
	// IHL = Internet Header Length, measured in 32-bit words
	// IHL of 5 means 5 × 4 bytes = 20 bytes (minimum IPv4 header, no options)
	// 0x45 = 0100 0101 in binary = Version 4, IHL 5
	header[0] = 0x45

	// Type of Service (ToS) / Differentiated Services Code Point (DSCP)
	// 0 = default/best-effort delivery, no special QoS or priority
	// Could be made configurable for QoS requirements in the future
	header[1] = 0

	// Total Length (16 bits)
	// Total size of the IP packet including header and data (in bytes)
	// 20 bytes (IP header) + UDP header (8 bytes) + UDP payload
	totalLen := 20 + dataLen
	binary.BigEndian.PutUint16(header[2:4], uint16(totalLen))

	// Identification (16 bits)
	// Used to identify fragments of an original IP datagram for reassembly
	// All fragments of the same packet share the same identification value
	// We use 0 since we're sending complete, unfragmented packets
	binary.BigEndian.PutUint16(header[4:6], 0)

	// Flags (3 bits) and Fragment Offset (13 bits)
	// Flags: [Reserved(1) | DF(1) | MF(1)]
	//   DF = Don't Fragment, MF = More Fragments
	// Fragment Offset = position of fragment in original packet (in 8-byte blocks)
	// 0 = no flags set, no fragmentation, offset 0
	binary.BigEndian.PutUint16(header[6:8], 0)

	// TTL - Time To Live (8 bits)
	// Number of hops (routers) the packet can traverse before being discarded
	// Decremented by 1 at each router; packet dropped when TTL reaches 0
	// 64 is a common default value (also used by Linux and many network devices)
	header[8] = 64

	// Protocol (8 bits)
	// Identifies the protocol of the encapsulated payload
	// 17 = UDP (see RFC 790 for protocol numbers)
	// Other common values: 1=ICMP, 6=TCP, 41=IPv6
	header[9] = 17

	// Header Checksum (16 bits)
	// Error-detection field for the IP header only (not the payload)
	// Initially set to 0, calculated after all other header fields are set
	// Uses one's complement sum of all 16-bit words in the header
	header[10] = 0
	header[11] = 0

	// Source IP Address (32 bits)
	// The spoofed source IP address for this packet
	// This is what allows IP address spoofing
	copy(header[12:16], srcIP.To4())

	// Destination IP Address (32 bits)
	// The target IP address where this packet will be sent
	copy(header[16:20], destIP.To4())

	// Calculate checksum
	checksum := calculateChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	return header
}

// buildIPv6Header constructs an IPv6 header
// IPv6 headers are simpler than IPv4: 40 bytes, no fragmentation, no checksum
func (s *UDPSender) buildIPv6Header(dataLen int, srcIP net.IP, destIP net.IP) []byte {
	header := make([]byte, 40)

	// Version (4 bits), Traffic Class (8 bits), Flow Label (20 bits)
	// Version = 6 for IPv6
	// Traffic Class = 0 (similar to IPv4 ToS, used for QoS/priority)
	// Flow Label = 0 (used to identify packets in the same flow for special handling)
	// 0x60000000 = 0110 0000 0000 0000 0000 0000 0000 0000 in binary
	//            = Version 6, Traffic Class 0, Flow Label 0
	binary.BigEndian.PutUint32(header[0:4], 0x60000000)

	// Payload Length (16 bits)
	// Length of the data following this header (UDP header + UDP payload)
	// Unlike IPv4's Total Length, this does NOT include the IPv6 header itself
	// IPv6 header is always 40 bytes, so it's implicit
	binary.BigEndian.PutUint16(header[4:6], uint16(dataLen))

	// Next Header (8 bits)
	// Identifies the type of header immediately following the IPv6 header
	// 17 = UDP (same protocol numbers as IPv4)
	// Other values: 6=TCP, 58=ICMPv6, 0=Hop-by-Hop Options, 43=Routing, 44=Fragment
	header[6] = 17

	// Hop Limit (8 bits)
	// Equivalent to IPv4's TTL (Time To Live)
	// Decremented by 1 at each router; packet dropped when it reaches 0
	// 64 is a common default (same as we use for IPv4)
	header[7] = 64

	// Source Address (128 bits / 16 bytes)
	// The spoofed source IPv6 address for this packet
	// IPv6 addresses are 4 times larger than IPv4 (128 bits vs 32 bits)
	copy(header[8:24], srcIP.To16())

	// Destination Address (128 bits / 16 bytes)
	// The target IPv6 address where this packet will be sent
	copy(header[24:40], destIP.To16())

	// Note: IPv6 deliberately removed the header checksum for performance
	// Error detection is handled by link layer (Ethernet) and transport layer (UDP/TCP)
	return header
}

// buildUDPHeader constructs a UDP header
// UDP headers are simple: only 8 bytes with 4 fields
func (s *UDPSender) buildUDPHeader(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16, isIPv6 bool) []byte {
	header := make([]byte, 8)

	// Source Port (16 bits)
	// The spoofed source port for this packet
	// This is what allows port spoofing
	// Port numbers: 0-65535 (0-1023 are well-known, 1024-49151 are registered)
	binary.BigEndian.PutUint16(header[0:2], srcPort)

	// Destination Port (16 bits)
	// The target port where this packet will be sent
	// Examples: 53=DNS, 80=HTTP, 443=HTTPS, 514=Syslog, 8080=HTTP-Alt
	binary.BigEndian.PutUint16(header[2:4], destPort)

	// Length (16 bits)
	// Total length of the UDP datagram: header (8 bytes) + payload
	// Minimum is 8 (header only), maximum is 65535 (theoretical)
	// Practical max is usually limited by IP packet size (65535 - 20 for IPv4)
	length := 8 + len(payload)
	binary.BigEndian.PutUint16(header[4:6], uint16(length))

	// Checksum (16 bits)
	// Error detection for UDP header and payload
	// For IPv4: optional (can be 0 to skip), but recommended
	// For IPv6: mandatory (cannot be 0)
	// Calculated using a pseudo-header (includes src/dst IPs) + UDP header + payload
	// Initially set to 0, then calculated and filled in
	binary.BigEndian.PutUint16(header[6:8], 0)

	// Calculate and set the UDP checksum
	// Uses pseudo-header (src IP, dst IP, protocol, length) + UDP header + payload
	checksum := s.calculateUDPChecksum(header, payload, srcIP, destIP, isIPv6)
	binary.BigEndian.PutUint16(header[6:8], checksum)

	return header
}

// calculateChecksum calculates the IP checksum (RFC 1071)
func calculateChecksum(data []byte) uint16 {
	sum := uint32(0)

	// Sum all 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Add remaining byte if odd length
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

// calculateUDPChecksum calculates UDP checksum including pseudo-header for IPv4 or IPv6
func (s *UDPSender) calculateUDPChecksum(udpHeader, payload []byte, srcIP net.IP, destIP net.IP, isIPv6 bool) uint16 {
	var pseudoHeader []byte

	if isIPv6 {
		// IPv6 pseudo-header (40 bytes)
		pseudoHeader = make([]byte, 40)

		// Source Address (16 bytes)
		copy(pseudoHeader[0:16], srcIP.To16())

		// Destination Address (16 bytes)
		copy(pseudoHeader[16:32], destIP.To16())

		// UDP Length (4 bytes)
		udpLength := uint32(len(udpHeader) + len(payload))
		binary.BigEndian.PutUint32(pseudoHeader[32:36], udpLength)

		// Zeros (3 bytes)
		pseudoHeader[36] = 0
		pseudoHeader[37] = 0
		pseudoHeader[38] = 0

		// Next Header = UDP (1 byte)
		pseudoHeader[39] = 17
	} else {
		// IPv4 pseudo-header (12 bytes)
		pseudoHeader = make([]byte, 12)

		// Source IP
		copy(pseudoHeader[0:4], srcIP.To4())

		// Destination IP
		copy(pseudoHeader[4:8], destIP.To4())

		// Zero
		pseudoHeader[8] = 0

		// Protocol (UDP = 17)
		pseudoHeader[9] = 17

		// UDP length
		udpLength := len(udpHeader) + len(payload)
		binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(udpLength))
	}

	// Concatenate pseudo-header, UDP header (with checksum = 0), and payload
	checksumData := make([]byte, 0, len(pseudoHeader)+len(udpHeader)+len(payload))
	checksumData = append(checksumData, pseudoHeader...)
	checksumData = append(checksumData, udpHeader...)
	checksumData = append(checksumData, payload...)

	return calculateChecksum(checksumData)
}
