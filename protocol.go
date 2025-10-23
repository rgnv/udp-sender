package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

// processInputStream reads packets from stdin using the binary protocol
// Binary protocol per packet:
//   - Magic:          3 bytes (0xC1 0x21 0xB1 for synchronization detection)
//   - Version:        1 byte (4 = IPv4, 6 = IPv6)
//   - Source IP:      4 bytes (IPv4) or 16 bytes (IPv6), network byte order
//   - Dest IP:        4 bytes (IPv4) or 16 bytes (IPv6), network byte order
//   - Source Port:    2 bytes (uint16, network byte order/big endian)
//   - Dest Port:      2 bytes (uint16, network byte order/big endian)
//   - Payload Length: 2 bytes (uint16, network byte order/big endian)
//   - Payload:        N bytes (variable length)
func processInputStream(logger *Logger, sender *UDPSender, input io.Reader) error {
	reader := bufio.NewReader(input)
	packetCount := 0
	bytesSent := uint64(0)

	logger.Info("Stream mode: reading packets from stdin", map[string]any{
		"protocol": "[Magic(3)][Version(1)][SrcIP(4/16)][DestIP(4/16)][SrcPort(2)][DestPort(2)][PayloadLen(2)][Payload(N)]",
	})

	for {
		// Read and validate magic number
		var magicBytes [3]byte
		n, err := io.ReadFull(reader, magicBytes[:])
		if err == io.EOF {
			// End of stream
			logger.Info("Stream complete", map[string]any{
				"packets_sent": packetCount,
				"bytes_sent":   bytesSent,
			})
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading magic bytes: %w (read %d bytes)", err, n)
		}

		// Validate magic number
		if magicBytes[0] != MagicByte1 || magicBytes[1] != MagicByte2 || magicBytes[2] != MagicByte3 {
			return fmt.Errorf("invalid magic number: got [0x%02X 0x%02X 0x%02X], expected [0x%02X 0x%02X 0x%02X] - stream may be misaligned",
				magicBytes[0], magicBytes[1], magicBytes[2], MagicByte1, MagicByte2, MagicByte3)
		}

		// Read version byte
		var versionByte [1]byte
		if _, err := io.ReadFull(reader, versionByte[:]); err != nil {
			return fmt.Errorf("reading version byte: %w", err)
		}
		version := versionByte[0]

		// Validate version
		if version != 4 && version != 6 {
			return fmt.Errorf("invalid IP version: %d (must be 4 or 6)", version)
		}

		// Read source IP based on version
		var srcIP string
		if version == 4 {
			// IPv4: 4 bytes
			var srcIPBytes [4]byte
			if _, err := io.ReadFull(reader, srcIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv4 source address: %w", err)
			}
			srcIP = net.IP(srcIPBytes[:]).String()
		} else {
			// IPv6: 16 bytes
			var srcIPBytes [16]byte
			if _, err := io.ReadFull(reader, srcIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv6 source address: %w", err)
			}
			srcIP = net.IP(srcIPBytes[:]).String()
		}

		// Read destination IP based on version (same size as source)
		var destIP string
		if version == 4 {
			// IPv4: 4 bytes
			var destIPBytes [4]byte
			if _, err := io.ReadFull(reader, destIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv4 destination address: %w", err)
			}
			destIP = net.IP(destIPBytes[:]).String()
		} else {
			// IPv6: 16 bytes
			var destIPBytes [16]byte
			if _, err := io.ReadFull(reader, destIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv6 destination address: %w", err)
			}
			destIP = net.IP(destIPBytes[:]).String()
		}

		// Read source port (2 bytes, big endian)
		var srcPortBytes [2]byte
		if _, err := io.ReadFull(reader, srcPortBytes[:]); err != nil {
			return fmt.Errorf("reading source port: %w", err)
		}
		srcPort := binary.BigEndian.Uint16(srcPortBytes[:])

		// Read destination port (2 bytes, big endian)
		var destPortBytes [2]byte
		if _, err := io.ReadFull(reader, destPortBytes[:]); err != nil {
			return fmt.Errorf("reading destination port: %w", err)
		}
		destPort := binary.BigEndian.Uint16(destPortBytes[:])

		// Read payload length (2 bytes, big endian)
		var lenBytes [2]byte
		if _, err := io.ReadFull(reader, lenBytes[:]); err != nil {
			return fmt.Errorf("reading payload length: %w", err)
		}
		payloadLen := binary.BigEndian.Uint16(lenBytes[:])

		// Read payload
		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := io.ReadFull(reader, payload); err != nil {
				return fmt.Errorf("reading payload (%d bytes): %w", payloadLen, err)
			}
		}

		// Send the packet
		n, err = sender.Send(string(payload), srcIP, strconv.Itoa(int(srcPort)), destIP, strconv.Itoa(int(destPort)))
		if err != nil {
			return fmt.Errorf("sending packet %d: %w", packetCount+1, err)
		}

		packetCount++
		bytesSent += uint64(n)

		// Progress feedback every 100 packets
		if packetCount%100 == 0 {
			logger.Debug("Progress update", map[string]any{
				"packets_sent": packetCount,
				"bytes_sent":   bytesSent,
			})
		}
	}
}
