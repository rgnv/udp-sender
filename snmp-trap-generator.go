//go:build ignore
// +build ignore

// SNMP Trap Generator - Creates SNMP trap packets for udp-sender
// Encodes SNMP trap PDUs and wraps them in the binary protocol for udp-sender.
// Run with: go run snmp-trap-generator.go [OPTIONS] | sudo ./udp-sender
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// protocol constants (redefined here since we can't import from main package)
const (
	GenMagicByte1 = 0xC1
	GenMagicByte2 = 0x21
	GenMagicByte3 = 0xB1
	GenFlagIPv6   = 0x01
)

// common OIDs
const (
	genOIDSysUpTime   = "1.3.6.1.2.1.1.3.0"
	genOIDSnmpTrapOID = "1.3.6.1.6.3.1.1.4.1.0"
	genOIDColdStart   = "1.3.6.1.6.3.1.1.5.1"
	genOIDEnterprise  = "1.3.6.1.4.1.99999"
	genOIDSysDescr    = "1.3.6.1.2.1.1.1.0"
	genOIDSysName     = "1.3.6.1.2.1.1.5.0"
)

func main() {
	count := flag.Int("count", 10, "Number of traps to generate")
	version := flag.String("version", "2c", "SNMP version: 1, 2c, 3")
	community := flag.String("community", "public", "Community string (v1/v2c)")
	baseIP := flag.String("base-ip", "10.0.0.1", "Base source IP (will increment)")
	basePort := flag.Int("base-port", 161, "Base source port")
	destIP := flag.String("dest-ip", "192.168.1.100", "Destination IP")
	destPort := flag.Int("dest-port", 162, "Destination port")
	trapOID := flag.String("trap-oid", genOIDColdStart, "Trap OID")
	enterprise := flag.String("enterprise", genOIDEnterprise, "Enterprise OID (v1)")
	securityName := flag.String("security-name", "", "SNMPv3 USM username")
	authProto := flag.String("auth-proto", "", "SNMPv3 auth protocol (MD5, SHA, SHA256)")
	authPass := flag.String("auth-pass", "", "SNMPv3 auth passphrase")
	privProto := flag.String("priv-proto", "", "SNMPv3 privacy protocol (DES, AES)")
	privPass := flag.String("priv-pass", "", "SNMPv3 privacy passphrase")
	ipv6 := flag.Bool("ipv6", false, "Generate IPv6 packets")
	message := flag.String("message", "SNMP trap from udp-sender", "Message in sysDescr varbind")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "SNMP Trap Generator - Creates SNMP trap packets for udp-sender\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] | sudo ./udp-sender\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Encodes SNMP trap PDUs and wraps them in udp-sender's binary protocol.\n")
		fmt.Fprintf(os.Stderr, "Pipe output directly to udp-sender -- same pattern as packet-generator.go.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// validate IPs
	baseIPAddr := net.ParseIP(*baseIP)
	if baseIPAddr == nil {
		log.Fatalf("Invalid base IP: %s", *baseIP)
	}
	destIPAddr := net.ParseIP(*destIP)
	if destIPAddr == nil {
		log.Fatalf("Invalid destination IP: %s", *destIP)
	}

	isIPv6 := baseIPAddr.To4() == nil
	if *ipv6 && !isIPv6 {
		log.Fatalf("IPv6 flag set but base IP is IPv4: %s", *baseIP)
	}
	if !*ipv6 && isIPv6 {
		log.Fatalf("IPv4 mode but base IP is IPv6: %s (use -ipv6 flag)", *baseIP)
	}

	destIsIPv6 := destIPAddr.To4() == nil
	if isIPv6 != destIsIPv6 {
		log.Fatalf("Source and dest IP versions must match (src: %s, dest: %s)", *baseIP, *destIP)
	}

	// v3 validation
	if strings.ToLower(*version) == "3" || strings.ToLower(*version) == "v3" {
		if *securityName == "" {
			log.Fatalf("SNMPv3 requires --security-name")
		}
	}

	fmt.Fprintf(os.Stderr, "Generating %d SNMPv%s traps: %s:%d -> %s:%d (oid: %s)\n",
		*count, *version, *baseIP, *basePort, *destIP, *destPort, *trapOID)

	for i := 0; i < *count; i++ {
		var srcIP net.IP
		if *ipv6 {
			srcIP = make(net.IP, net.IPv6len)
			copy(srcIP, baseIPAddr.To16())
			srcIP[15] = byte((int(baseIPAddr.To16()[15]) + i) % 256)
		} else {
			srcIP = make(net.IP, net.IPv4len)
			copy(srcIP, baseIPAddr.To4())
			srcIP[3] = byte((int(baseIPAddr.To4()[3]) + i) % 256)
		}
		srcPort := uint16(*basePort + i)

		writeBinaryTrap(srcIP, srcPort, destIPAddr, uint16(*destPort), *version, *community,
			*trapOID, *enterprise, *securityName, *authProto, *authPass, *privProto, *privPass, *message, i, isIPv6)

		if (i+1)%100 == 0 {
			fmt.Fprintf(os.Stderr, "Generated %d traps...\n", i+1)
		}
	}

	fmt.Fprintf(os.Stderr, "Complete: generated %d traps\n", *count)
}

func writeBinaryTrap(srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16,
	version, community, trapOID, enterprise, secName, authProto, authPass, privProto, privPass, message string, seq int, isIPv6 bool) {

	// build the SNMP trap PDU bytes
	var pduBytes []byte
	var err error

	timestamp := uint32(time.Now().Unix())

	varbinds := []g.SnmpPDU{
		{Name: genOIDSysDescr, Type: g.OctetString, Value: fmt.Sprintf("%s #%d", message, seq+1)},
		{Name: genOIDSysName, Type: g.OctetString, Value: "udp-sender"},
	}

	switch strings.ToLower(version) {
	case "1", "v1":
		allPDUs := varbinds
		packet := &g.SnmpPacket{
			Version:   g.Version1,
			Community: community,
			PDUType:   g.Trap,
			Variables: allPDUs,
			SnmpTrap: g.SnmpTrap{
				Variables:    allPDUs,
				Enterprise:   enterprise,
				AgentAddress: srcIP.String(),
				GenericTrap:  6,
				SpecificTrap: seq + 1,
				Timestamp:    uint(timestamp),
			},
		}
		pduBytes, err = packet.MarshalMsg()

	case "2c", "v2c", "2":
		allPDUs := make([]g.SnmpPDU, 0, len(varbinds)+2)
		allPDUs = append(allPDUs, g.SnmpPDU{Name: genOIDSysUpTime, Type: g.TimeTicks, Value: timestamp})
		allPDUs = append(allPDUs, g.SnmpPDU{Name: genOIDSnmpTrapOID, Type: g.ObjectIdentifier, Value: trapOID})
		allPDUs = append(allPDUs, varbinds...)
		packet := &g.SnmpPacket{
			Version:   g.Version2c,
			Community: community,
			PDUType:   g.SNMPv2Trap,
			Variables: allPDUs,
			RequestID: uint32(time.Now().UnixNano() & 0x7FFFFFFF),
		}
		pduBytes, err = packet.MarshalMsg()

	case "3", "v3":
		allPDUs := make([]g.SnmpPDU, 0, len(varbinds)+2)
		allPDUs = append(allPDUs, g.SnmpPDU{Name: genOIDSysUpTime, Type: g.TimeTicks, Value: timestamp})
		allPDUs = append(allPDUs, g.SnmpPDU{Name: genOIDSnmpTrapOID, Type: g.ObjectIdentifier, Value: trapOID})
		allPDUs = append(allPDUs, varbinds...)

		var msgFlags g.SnmpV3MsgFlags
		ap := parseGenAuthProto(authProto)
		pp := parseGenPrivProto(privProto)
		switch {
		case ap != g.NoAuth && pp != g.NoPriv:
			msgFlags = g.AuthPriv
		case ap != g.NoAuth:
			msgFlags = g.AuthNoPriv
		default:
			msgFlags = g.NoAuthNoPriv
		}

		usmParams := &g.UsmSecurityParameters{
			UserName:                 secName,
			AuthenticationProtocol:   ap,
			AuthenticationPassphrase: authPass,
			PrivacyProtocol:          pp,
			PrivacyPassphrase:        privPass,
			AuthoritativeEngineID:    "udp-sender",
		}
		if err := usmParams.InitSecurityKeys(); err != nil {
			log.Fatalf("Failed to init SNMPv3 security keys: %v", err)
		}

		packet := &g.SnmpPacket{
			Version:            g.Version3,
			PDUType:            g.SNMPv2Trap,
			MsgFlags:           msgFlags,
			SecurityModel:      g.UserSecurityModel,
			SecurityParameters: usmParams,
			ContextEngineID:    "udp-sender",
			Variables:          allPDUs,
			RequestID:          uint32(time.Now().UnixNano() & 0x7FFFFFFF),
			MsgID:              uint32(time.Now().UnixNano() & 0x7FFFFFFF),
			MsgMaxSize:         65507,
		}
		pduBytes, err = packet.MarshalMsg()

	default:
		log.Fatalf("Unsupported SNMP version: %s", version)
	}

	if err != nil {
		log.Fatalf("Error encoding SNMP trap %d: %v", seq+1, err)
	}

	// write binary protocol frame
	// magic
	os.Stdout.Write([]byte{GenMagicByte1, GenMagicByte2, GenMagicByte3})

	// flags
	var flags byte
	if isIPv6 {
		flags |= GenFlagIPv6
	}
	os.Stdout.Write([]byte{flags})

	// src IP
	if isIPv6 {
		os.Stdout.Write(srcIP.To16())
	} else {
		os.Stdout.Write(srcIP.To4())
	}

	// dest IP
	if isIPv6 {
		os.Stdout.Write(destIP.To16())
	} else {
		os.Stdout.Write(destIP.To4())
	}

	// src port
	var portBuf [2]byte
	binary.BigEndian.PutUint16(portBuf[:], srcPort)
	os.Stdout.Write(portBuf[:])

	// dest port
	binary.BigEndian.PutUint16(portBuf[:], destPort)
	os.Stdout.Write(portBuf[:])

	// payload length
	var lenBuf [2]byte
	binary.BigEndian.PutUint16(lenBuf[:], uint16(len(pduBytes)))
	os.Stdout.Write(lenBuf[:])

	// payload
	os.Stdout.Write(pduBytes)
}

func parseGenAuthProto(proto string) g.SnmpV3AuthProtocol {
	switch strings.ToUpper(proto) {
	case "MD5":
		return g.MD5
	case "SHA", "SHA1":
		return g.SHA
	case "SHA224":
		return g.SHA224
	case "SHA256":
		return g.SHA256
	case "SHA384":
		return g.SHA384
	case "SHA512":
		return g.SHA512
	default:
		return g.NoAuth
	}
}

func parseGenPrivProto(proto string) g.SnmpV3PrivProtocol {
	switch strings.ToUpper(proto) {
	case "DES":
		return g.DES
	case "AES", "AES128":
		return g.AES
	case "AES192":
		return g.AES192
	case "AES256":
		return g.AES256
	default:
		return g.NoPriv
	}
}
