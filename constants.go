package main

// Protocol magic number for packet synchronization and stream alignment detection
const (
	MagicByte1 = 0xC1
	MagicByte2 = 0x21
	MagicByte3 = 0xB1
)

// Protocol flags bitfield
// Bit 0: IP version flag (0 = IPv4, 1 = IPv6)
// Bits 1-7: reserved for future use
const (
	FlagIPv6 = 0x01
)

// MTU (Maximum Transmission Unit) constants
const (
	// DefaultMTU is the standard Ethernet MTU (1500 bytes)
	DefaultMTU = 1500

	// MinMTU is the minimum allowed MTU (576 bytes - minimum IPv4 MTU)
	MinMTU = 576

	// MaxMTU is the maximum allowed MTU (9000 bytes - jumbo frames)
	MaxMTU = 9000

	// IPv4HeaderSize is the size of an IPv4 header
	IPv4HeaderSize = 20

	// IPv6HeaderSize is the size of an IPv6 header
	IPv6HeaderSize = 40

	// UDPHeaderSize is the size of a UDP header
	UDPHeaderSize = 8
)

// Default maximum payload sizes based on standard Ethernet MTU (1500 bytes)
// These are kept for backward compatibility and reference
const (
	// MaxPayloadIPv4: 1500 - 20 (IP header) - 8 (UDP header) = 1472 bytes max payload
	MaxPayloadIPv4 = 1472

	// MaxPayloadIPv6: 1500 - 40 (IPv6 header) - 8 (UDP header) = 1452 bytes max payload
	MaxPayloadIPv6 = 1452
)

// SNMP trap constants
const (
	// DefaultSNMPTrapPort is the standard SNMP trap receiver port
	DefaultSNMPTrapPort = 162

	// DefaultSNMPCommunity is the default community string for v1/v2c
	DefaultSNMPCommunity = "public"

	// DefaultSNMPEngineID is a default engine ID for v3 traps
	DefaultSNMPEngineID = "udp-sender"
)

// Common SNMP OIDs
const (
	OIDSysUpTime    = "1.3.6.1.2.1.1.3.0"
	OIDSnmpTrapOID  = "1.3.6.1.6.3.1.1.4.1.0"
	OIDColdStart    = "1.3.6.1.6.3.1.1.5.1"
	OIDWarmStart    = "1.3.6.1.6.3.1.1.5.2"
	OIDLinkDown     = "1.3.6.1.6.3.1.1.5.3"
	OIDLinkUp       = "1.3.6.1.6.3.1.1.5.4"
	OIDAuthFailure  = "1.3.6.1.6.3.1.1.5.5"
	OIDEnterprise   = "1.3.6.1.4.1.99999"
	OIDSysDescr     = "1.3.6.1.2.1.1.1.0"
	OIDSysObjectID  = "1.3.6.1.2.1.1.2.0"
	OIDSysContact   = "1.3.6.1.2.1.1.4.0"
	OIDSysName      = "1.3.6.1.2.1.1.5.0"
	OIDSysLocation  = "1.3.6.1.2.1.1.6.0"
)
