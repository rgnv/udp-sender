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

	// Version (4) and IHL (5 = 20 bytes)
	header[0] = 0x45

	// Type of Service
	header[1] = 0

	// Total Length
	totalLen := 20 + dataLen
	binary.BigEndian.PutUint16(header[2:4], uint16(totalLen))

	// Identification
	binary.BigEndian.PutUint16(header[4:6], 0)

	// Flags and Fragment Offset
	binary.BigEndian.PutUint16(header[6:8], 0)

	// TTL
	header[8] = 64

	// Protocol (UDP = 17)
	header[9] = 17

	// Checksum (will be calculated)
	header[10] = 0
	header[11] = 0

	// Source IP
	copy(header[12:16], srcIP.To4())

	// Destination IP
	copy(header[16:20], destIP.To4())

	// Calculate checksum
	checksum := calculateChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	return header
}

// buildIPv6Header constructs an IPv6 header
func (s *UDPSender) buildIPv6Header(dataLen int, srcIP net.IP, destIP net.IP) []byte {
	header := make([]byte, 40)

	// Version (6), Traffic Class (0), Flow Label (0)
	binary.BigEndian.PutUint32(header[0:4], 0x60000000) // Version 6

	// Payload Length (UDP header + payload)
	binary.BigEndian.PutUint16(header[4:6], uint16(dataLen))

	// Next Header (UDP = 17)
	header[6] = 17

	// Hop Limit
	header[7] = 64

	// Source Address (16 bytes)
	copy(header[8:24], srcIP.To16())

	// Destination Address (16 bytes)
	copy(header[24:40], destIP.To16())

	// IPv6 doesn't have a header checksum
	return header
}

// buildUDPHeader constructs a UDP header
func (s *UDPSender) buildUDPHeader(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16, isIPv6 bool) []byte {
	header := make([]byte, 8)

	// Source port
	binary.BigEndian.PutUint16(header[0:2], srcPort)

	// Destination port
	binary.BigEndian.PutUint16(header[2:4], destPort)

	// Length (header + payload)
	length := 8 + len(payload)
	binary.BigEndian.PutUint16(header[4:6], uint16(length))

	// Checksum (0 = no checksum for UDP)
	binary.BigEndian.PutUint16(header[6:8], 0)

	// Calculate UDP checksum (optional but recommended)
	checksum := s.calculateUDPChecksum(header, payload, srcIP, destIP, isIPv6)
	binary.BigEndian.PutUint16(header[6:8], checksum)

	return header
}

// calculateChecksum calculates the Internet checksum (RFC 1071)
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
