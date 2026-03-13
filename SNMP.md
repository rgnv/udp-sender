# SNMP Trap Support

udp-sender supports sending SNMP trap packets (v1, v2c, v3) with full IP/port spoofing via raw sockets. This works exactly like syslog or any other UDP payload -- udp-sender is payload-agnostic. The `snmp-trap-generator` encodes SNMP trap PDUs and wraps them in udp-sender's binary protocol. udp-sender just sends the bytes.

This is useful for testing SNMP trap receivers like Cribl Stream, Splunk, or any NMS that accepts traps on UDP port 162.

## How It Works

Same flow as syslog or any other UDP data:

1. `snmp-trap-generator` encodes SNMP trap PDUs (v1/v2c/v3) using gosnmp
2. Wraps each PDU in the binary protocol frame (magic + flags + src/dst IPs + ports + payload)
3. Writes to stdout
4. `udp-sender` reads the binary stream, constructs raw IP+UDP packets with spoofed source, sends

```
snmp-trap-generator -> [binary protocol on stdout] -> udp-sender -> [raw socket] -> network
```

udp-sender never parses or understands SNMP. It just sends whatever payload bytes it gets.

## Examples

```bash
# v2c coldStart traps to a receiver
go run snmp-trap-generator.go -version 2c -count 100 \
  -dest-ip 192.168.1.100 -dest-port 162 | sudo ./udp-sender

# v1 traps with enterprise OID
go run snmp-trap-generator.go -version 1 -count 50 \
  -dest-ip 192.168.1.100 -dest-port 162 \
  -enterprise "1.3.6.1.4.1.99999" | sudo ./udp-sender

# v3 traps with SHA auth and AES encryption
go run snmp-trap-generator.go -version 3 -count 10 \
  -dest-ip 192.168.1.100 -dest-port 162 \
  -security-name myuser -auth-proto SHA -auth-pass "myauthpass123456" \
  -priv-proto AES -priv-pass "myprivpass123456" | sudo ./udp-sender

# spoofed source IPs (10.0.0.1, 10.0.0.2, ... incrementing)
go run snmp-trap-generator.go -version 2c -count 50 \
  -base-ip 10.0.0.1 -base-port 161 \
  -dest-ip 192.168.1.100 -dest-port 162 | sudo ./udp-sender

# save to file, replay later
go run snmp-trap-generator.go -version 2c -count 1000 \
  -dest-ip 192.168.1.100 > snmp-traps.bin
cat snmp-traps.bin | sudo ./udp-sender

# jumbo frames
go run snmp-trap-generator.go -version 2c -count 100 \
  -dest-ip 192.168.1.100 | sudo ./udp-sender -m 9000
```

## snmp-trap-generator Flags

```
Usage: snmp-trap-generator [OPTIONS]

Options:
  -count int          Number of traps to generate (default 10)
  -version string     SNMP version: 1, 2c, 3 (default "2c")
  -community string   Community string (default "public")
  -base-ip string     Base source IP, will increment (default "10.0.0.1")
  -base-port int      Base source port (default 161)
  -dest-ip string     Destination IP (default "192.168.1.100")
  -dest-port int      Destination port (default 162)
  -trap-oid string    Trap OID (default coldStart)
  -enterprise string  Enterprise OID for v1 (default "1.3.6.1.4.1.99999")
  -security-name      SNMPv3 USM username
  -auth-proto         SNMPv3 auth protocol (MD5, SHA, SHA256, etc.)
  -auth-pass          SNMPv3 auth passphrase
  -priv-proto         SNMPv3 privacy protocol (DES, AES, etc.)
  -priv-pass          SNMPv3 privacy passphrase
  -message string     Message in sysDescr varbind (default "SNMP trap from udp-sender")
  -ipv6               Generate IPv6 packets
  -json               Output JSON lines for --snmp mode instead of binary protocol
```

## Common Trap OIDs

| OID | Name | Description |
|-----|------|-------------|
| 1.3.6.1.6.3.1.1.5.1 | coldStart | Agent reinitializing, config may change |
| 1.3.6.1.6.3.1.1.5.2 | warmStart | Agent reinitializing, config unchanged |
| 1.3.6.1.6.3.1.1.5.3 | linkDown | Network interface went down |
| 1.3.6.1.6.3.1.1.5.4 | linkUp | Network interface came up |
| 1.3.6.1.6.3.1.1.5.5 | authenticationFailure | SNMP auth failure |

## Dependencies

SNMP trap encoding uses [gosnmp](https://github.com/gosnmp/gosnmp) (BSD license) for ASN.1/BER encoding and SNMPv3 USM (auth + encryption).
