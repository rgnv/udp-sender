package main

import (
	"fmt"
	"net"
	"strconv"
	"syscall"
)

// PacketSender is an interface for sending UDP packets
type PacketSender interface {
	Send(message string, srcHost string, srcPort string, destHost string, destPort string) (int, error)
	Close() error
}

// UDPSender handles UDP packet sending with raw sockets for IP spoofing.
// This class implements the PacketSender interface. Both source and destination
// IP and port are specified per-packet in the Send() method.
// Supports both IPv4 and IPv6.
type UDPSender struct {
	fdIPv4 int
	fdIPv6 int
}

// Ensure UDPSender implements PacketSender interface at compile time
var _ PacketSender = (*UDPSender)(nil)

// NewUDPSender creates a new UDP sender with raw socket support
// Both source and destination IP and port are specified per-packet in the Send() method
// Requires root/admin privileges to create raw sockets
// Supports both IPv4 and IPv6
func NewUDPSender() (*UDPSender, error) {
	sender := &UDPSender{
		fdIPv4: -1,
		fdIPv6: -1,
	}

	// Create IPv4 socket
	fd4, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw IPv4 socket (requires root): %w", err)
	}

	// Set IP_HDRINCL option to tell the kernel we're providing the IP header
	err = syscall.SetsockoptInt(fd4, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		_ = syscall.Close(fd4)
		return nil, fmt.Errorf("failed to set IP_HDRINCL: %w", err)
	}

	sender.fdIPv4 = fd4

	// Create IPv6 socket
	fd6, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		// Clean up IPv4 socket
		_ = syscall.Close(sender.fdIPv4)
		return nil, fmt.Errorf("failed to create raw IPv6 socket (requires root): %w", err)
	}
	// Note: IPv6 raw sockets don't require IPV6_HDRINCL on most systems
	// The kernel automatically sets the next header field

	sender.fdIPv6 = fd6

	return sender, nil
}

// Send sends a message via raw UDP socket with spoofed source and destination
// Implements PacketSender.Send
// srcHost and srcPort specify the spoofed source address for this packet
// destHost and destPort specify the destination address for this packet
func (s *UDPSender) Send(message string, srcHost string, srcPort string, destHost string, destPort string) (int, error) {
	payload := []byte(message)

	// Resolve source address
	srcIPs, err := net.LookupIP(srcHost)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve source host: %w", err)
	}
	if len(srcIPs) == 0 {
		return 0, fmt.Errorf("no IP addresses found for source host: %s", srcHost)
	}

	// Resolve destination address
	destIPs, err := net.LookupIP(destHost)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve destination host: %w", err)
	}
	if len(destIPs) == 0 {
		return 0, fmt.Errorf("no IP addresses found for destination host: %s", destHost)
	}

	// Separate IPv4 and IPv6 addresses
	var srcIPv4, srcIPv6, destIPv4, destIPv6 net.IP
	for _, ip := range srcIPs {
		if ip.To4() != nil && srcIPv4 == nil {
			srcIPv4 = ip.To4()
		} else if ip.To16() != nil && ip.To4() == nil && srcIPv6 == nil {
			srcIPv6 = ip.To16()
		}
	}
	for _, ip := range destIPs {
		if ip.To4() != nil && destIPv4 == nil {
			destIPv4 = ip.To4()
		} else if ip.To16() != nil && ip.To4() == nil && destIPv6 == nil {
			destIPv6 = ip.To16()
		}
	}

	// Parse ports
	srcPortNum, err := strconv.Atoi(srcPort)
	if err != nil || srcPortNum < 0 || srcPortNum > 65535 {
		return 0, fmt.Errorf("invalid source port: %s", srcPort)
	}
	srcPortUint := uint16(srcPortNum)

	destPortNum, err := strconv.Atoi(destPort)
	if err != nil || destPortNum < 0 || destPortNum > 65535 {
		return 0, fmt.Errorf("invalid destination port: %s", destPort)
	}
	destPortUint := uint16(destPortNum)

	// Determine which IP family to use - prefer IPv4 if both are available
	if srcIPv4 != nil && destIPv4 != nil {
		// Use IPv4
		packet := s.buildPacket(payload, srcIPv4, srcPortUint, destIPv4, destPortUint)

		addr4 := &syscall.SockaddrInet4{
			Port: int(destPortUint),
		}
		copy(addr4.Addr[:], destIPv4.To4())

		err = syscall.Sendto(s.fdIPv4, packet, 0, addr4)
		if err != nil {
			return 0, fmt.Errorf("failed to send packet to %s: %w", destIPv4, err)
		}

		return len(payload), nil
	}

	if srcIPv6 != nil && destIPv6 != nil {
		// Use IPv6
		packet := s.buildPacket(payload, srcIPv6, srcPortUint, destIPv6, destPortUint)

		addr6 := &syscall.SockaddrInet6{
			Port: int(destPortUint),
		}
		copy(addr6.Addr[:], destIPv6.To16())

		err = syscall.Sendto(s.fdIPv6, packet, 0, addr6)
		if err != nil {
			return 0, fmt.Errorf("failed to send packet to %s: %w", destIPv6, err)
		}

		return len(payload), nil
	}

	// No compatible source/destination combination
	return 0, fmt.Errorf("no compatible source/destination IP combination available (src: %s, dest: %s)", srcHost, destHost)
}

// Close closes the raw socket(s)
// Implements PacketSender.Close
func (s *UDPSender) Close() error {
	var err4, err6 error

	if s.fdIPv4 >= 0 {
		err4 = syscall.Close(s.fdIPv4)
	}
	if s.fdIPv6 >= 0 {
		err6 = syscall.Close(s.fdIPv6)
	}

	// Return first error encountered
	if err4 != nil {
		return err4
	}
	return err6
}
