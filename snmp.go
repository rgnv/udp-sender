package main

import (
	"fmt"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// initSecurityKeysFn is the function used to initialize USM security keys.
// swappable for testing the error path since gosnmp's InitSecurityKeys
// practically never fails with standard crypto.
var initSecurityKeysFn = func(usmParams *g.UsmSecurityParameters) error {
	return usmParams.InitSecurityKeys()
}

// SNMPVarbind represents a single variable binding for SNMP traps
type SNMPVarbind struct {
	OID   string
	Type  g.Asn1BER
	Value interface{}
}

// SNMPv3SecurityParams holds SNMPv3 USM security configuration
type SNMPv3SecurityParams struct {
	UserName       string
	AuthProtocol   g.SnmpV3AuthProtocol
	AuthPassphrase string
	PrivProtocol   g.SnmpV3PrivProtocol
	PrivPassphrase string
	EngineID       string
	EngineBoots    uint32
	EngineTime     uint32
}

// buildSNMPv1TrapPDU encodes an SNMPv1 Trap PDU as raw bytes
func buildSNMPv1TrapPDU(community, enterprise, agentAddr string, genericTrap, specificTrap int, timestamp uint, varbinds []SNMPVarbind) ([]byte, error) {
	if enterprise == "" {
		return nil, fmt.Errorf("SNMPv1 trap requires an enterprise OID")
	}
	if agentAddr == "" {
		return nil, fmt.Errorf("SNMPv1 trap requires an agent address")
	}

	pdus := snmpVarbindsToPDUs(varbinds)

	packet := &g.SnmpPacket{
		Version:   g.Version1,
		Community: community,
		PDUType:   g.Trap,
		Variables: pdus,
		SnmpTrap: g.SnmpTrap{
			Enterprise:   enterprise,
			AgentAddress: agentAddr,
			GenericTrap:  genericTrap,
			SpecificTrap: specificTrap,
			Timestamp:    timestamp,
		},
	}

	return packet.MarshalMsg()
}

// buildSNMPv2cTrapPDU encodes an SNMPv2c Trap PDU as raw bytes
func buildSNMPv2cTrapPDU(community, trapOID string, timestamp uint32, varbinds []SNMPVarbind) ([]byte, error) {
	if trapOID == "" {
		return nil, fmt.Errorf("SNMPv2c trap requires a trap OID")
	}

	// v2c traps need sysUpTime and snmpTrapOID as first two varbinds
	pdus := make([]g.SnmpPDU, 0, len(varbinds)+2)

	// sysUpTime.0
	if timestamp == 0 {
		timestamp = uint32(time.Now().Unix()) //nolint:gosec
	}
	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSysUpTime,
		Type:  g.TimeTicks,
		Value: timestamp,
	})

	// snmpTrapOID.0
	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSnmpTrapOID,
		Type:  g.ObjectIdentifier,
		Value: trapOID,
	})

	// user varbinds
	pdus = append(pdus, snmpVarbindsToPDUs(varbinds)...)

	packet := &g.SnmpPacket{
		Version:   g.Version2c,
		Community: community,
		PDUType:   g.SNMPv2Trap,
		Variables: pdus,
		RequestID: uint32(time.Now().UnixNano() & 0x7FFFFFFF), //nolint:gosec
	}

	return packet.MarshalMsg()
}

// buildSNMPv3TrapPDU encodes an SNMPv3 Trap PDU as raw bytes
func buildSNMPv3TrapPDU(secParams SNMPv3SecurityParams, trapOID string, timestamp uint32, varbinds []SNMPVarbind) ([]byte, error) {
	if trapOID == "" {
		return nil, fmt.Errorf("SNMPv3 trap requires a trap OID")
	}
	if secParams.UserName == "" {
		return nil, fmt.Errorf("SNMPv3 trap requires a security username")
	}

	// v3 traps have same varbind structure as v2c
	pdus := make([]g.SnmpPDU, 0, len(varbinds)+2)

	if timestamp == 0 {
		timestamp = uint32(time.Now().Unix()) //nolint:gosec
	}
	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSysUpTime,
		Type:  g.TimeTicks,
		Value: timestamp,
	})

	pdus = append(pdus, g.SnmpPDU{
		Name:  OIDSnmpTrapOID,
		Type:  g.ObjectIdentifier,
		Value: trapOID,
	})

	pdus = append(pdus, snmpVarbindsToPDUs(varbinds)...)

	// figure out msg flags based on auth/priv config
	var msgFlags g.SnmpV3MsgFlags
	switch {
	case secParams.AuthProtocol != g.NoAuth && secParams.PrivProtocol != g.NoPriv:
		msgFlags = g.AuthPriv
	case secParams.AuthProtocol != g.NoAuth:
		msgFlags = g.AuthNoPriv
	default:
		msgFlags = g.NoAuthNoPriv
	}

	engineID := secParams.EngineID
	if engineID == "" {
		engineID = DefaultSNMPEngineID
	}

	usmParams := &g.UsmSecurityParameters{
		UserName:                 secParams.UserName,
		AuthenticationProtocol:   secParams.AuthProtocol,
		AuthenticationPassphrase: secParams.AuthPassphrase,
		PrivacyProtocol:          secParams.PrivProtocol,
		PrivacyPassphrase:        secParams.PrivPassphrase,
		AuthoritativeEngineID:    engineID,
		AuthoritativeEngineBoots: secParams.EngineBoots,
		AuthoritativeEngineTime:  secParams.EngineTime,
	}

	if err := initSecurityKeysFn(usmParams); err != nil {
		return nil, fmt.Errorf("failed to init SNMPv3 security keys: %w", err)
	}

	packet := &g.SnmpPacket{
		Version:            g.Version3,
		PDUType:            g.SNMPv2Trap,
		MsgFlags:           msgFlags,
		SecurityModel:      g.UserSecurityModel,
		SecurityParameters: usmParams,
		ContextEngineID:    engineID,
		Variables:          pdus,
		RequestID:          uint32(time.Now().UnixNano() & 0x7FFFFFFF), //nolint:gosec
		MsgID:              uint32(time.Now().UnixNano() & 0x7FFFFFFF), //nolint:gosec
		MsgMaxSize:         65507,
	}

	return packet.MarshalMsg()
}

// snmpVarbindsToPDUs converts our varbind type to gosnmp PDUs
func snmpVarbindsToPDUs(varbinds []SNMPVarbind) []g.SnmpPDU {
	pdus := make([]g.SnmpPDU, 0, len(varbinds))
	for _, vb := range varbinds {
		pdus = append(pdus, g.SnmpPDU{
			Name:  vb.OID,
			Type:  vb.Type,
			Value: vb.Value,
		})
	}
	return pdus
}

// defaultVarbinds returns a standard set of varbinds for test/demo traps
func defaultVarbinds() []SNMPVarbind {
	return []SNMPVarbind{
		{OID: OIDSysDescr, Type: g.OctetString, Value: "udp-sender SNMP trap"},
		{OID: OIDSysName, Type: g.OctetString, Value: "udp-sender"},
	}
}
