package main

import (
	"fmt"
	"strings"
	"testing"

	g "github.com/gosnmp/gosnmp"
)

func TestBuildSNMPv1TrapPDU(t *testing.T) {
	t.Run("basic v1 trap", func(t *testing.T) {
		varbinds := []SNMPVarbind{
			{OID: OIDSysDescr, Type: g.OctetString, Value: "test trap"},
		}
		data, err := buildSNMPv1TrapPDU("public", OIDEnterprise, "10.0.0.1", 6, 1, 12345, varbinds)
		if err != nil {
			t.Fatalf("buildSNMPv1TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("buildSNMPv1TrapPDU() returned empty bytes")
		}
		// first byte should be SEQUENCE (0x30)
		if data[0] != 0x30 {
			t.Errorf("expected SEQUENCE tag 0x30, got 0x%02X", data[0])
		}
	})

	t.Run("v1 trap missing enterprise", func(t *testing.T) {
		_, err := buildSNMPv1TrapPDU("public", "", "10.0.0.1", 0, 0, 0, nil)
		if err == nil {
			t.Fatal("expected error for missing enterprise OID")
		}
	})

	t.Run("v1 trap missing agent address", func(t *testing.T) {
		_, err := buildSNMPv1TrapPDU("public", OIDEnterprise, "", 0, 0, 0, nil)
		if err == nil {
			t.Fatal("expected error for missing agent address")
		}
	})

	t.Run("v1 trap with empty varbinds", func(t *testing.T) {
		data, err := buildSNMPv1TrapPDU("public", OIDEnterprise, "10.0.0.1", 0, 0, 100, nil)
		if err != nil {
			t.Fatalf("buildSNMPv1TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty bytes")
		}
	})

	t.Run("v1 trap with multiple varbinds", func(t *testing.T) {
		varbinds := []SNMPVarbind{
			{OID: OIDSysDescr, Type: g.OctetString, Value: "test device"},
			{OID: OIDSysName, Type: g.OctetString, Value: "router-1"},
			{OID: OIDSysLocation, Type: g.OctetString, Value: "datacenter"},
		}
		data, err := buildSNMPv1TrapPDU("private", OIDEnterprise, "192.168.1.1", 6, 42, 99999, varbinds)
		if err != nil {
			t.Fatalf("buildSNMPv1TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty bytes")
		}
	})

	t.Run("v1 trap with IPv6 agent address", func(t *testing.T) {
		data, err := buildSNMPv1TrapPDU("public", OIDEnterprise, "2001:db8::1", 6, 1, 12345, nil)
		if err != nil {
			t.Fatalf("buildSNMPv1TrapPDU() with IPv6 agent error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty bytes for IPv6 agent")
		}
		if data[0] != 0x30 {
			t.Errorf("expected SEQUENCE tag 0x30, got 0x%02X", data[0])
		}
	})
}

func TestBuildSNMPv2cTrapPDU(t *testing.T) {
	t.Run("basic v2c trap", func(t *testing.T) {
		varbinds := []SNMPVarbind{
			{OID: OIDSysDescr, Type: g.OctetString, Value: "test trap v2c"},
		}
		data, err := buildSNMPv2cTrapPDU("public", OIDColdStart, 12345, varbinds)
		if err != nil {
			t.Fatalf("buildSNMPv2cTrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
		if data[0] != 0x30 {
			t.Errorf("expected SEQUENCE tag 0x30, got 0x%02X", data[0])
		}
	})

	t.Run("v2c trap missing OID", func(t *testing.T) {
		_, err := buildSNMPv2cTrapPDU("public", "", 0, nil)
		if err == nil {
			t.Fatal("expected error for missing trap OID")
		}
	})

	t.Run("v2c trap with zero timestamp", func(t *testing.T) {
		varbinds := []SNMPVarbind{
			{OID: OIDSysDescr, Type: g.OctetString, Value: "auto-timestamp"},
		}
		data, err := buildSNMPv2cTrapPDU("public", OIDWarmStart, 0, varbinds)
		if err != nil {
			t.Fatalf("buildSNMPv2cTrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
	})

	t.Run("v2c trap with different trap OIDs", func(t *testing.T) {
		oids := []string{OIDColdStart, OIDWarmStart, OIDLinkDown, OIDLinkUp, OIDAuthFailure}
		for _, oid := range oids {
			data, err := buildSNMPv2cTrapPDU("public", oid, 100, defaultVarbinds())
			if err != nil {
				t.Fatalf("buildSNMPv2cTrapPDU(%s) error: %v", oid, err)
			}
			if len(data) == 0 {
				t.Fatalf("buildSNMPv2cTrapPDU(%s) returned empty", oid)
			}
		}
	})

	t.Run("v2c trap with empty community", func(t *testing.T) {
		data, err := buildSNMPv2cTrapPDU("", OIDColdStart, 100, defaultVarbinds())
		if err != nil {
			t.Fatalf("buildSNMPv2cTrapPDU() with empty community error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty bytes with empty community")
		}
	})
}

func TestBuildSNMPv3TrapPDU(t *testing.T) {
	t.Run("v3 noauth nopriv", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:     "testuser",
			AuthProtocol: g.NoAuth,
			PrivProtocol: g.NoPriv,
			EngineID:     "test-engine",
		}
		varbinds := []SNMPVarbind{
			{OID: OIDSysDescr, Type: g.OctetString, Value: "v3 trap noauth"},
		}
		data, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 12345, varbinds)
		if err != nil {
			t.Fatalf("buildSNMPv3TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
		if data[0] != 0x30 {
			t.Errorf("expected SEQUENCE tag 0x30, got 0x%02X", data[0])
		}
	})

	t.Run("v3 auth nopriv MD5", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:       "authuser",
			AuthProtocol:   g.MD5,
			AuthPassphrase: "authpass12345678",
			PrivProtocol:   g.NoPriv,
			EngineID:       "test-engine",
		}
		data, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 100, defaultVarbinds())
		if err != nil {
			t.Fatalf("buildSNMPv3TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
	})

	t.Run("v3 auth priv SHA/AES", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:       "privuser",
			AuthProtocol:   g.SHA,
			AuthPassphrase: "authpass12345678",
			PrivProtocol:   g.AES,
			PrivPassphrase: "privpass12345678",
			EngineID:       "test-engine",
		}
		data, err := buildSNMPv3TrapPDU(secParams, OIDLinkDown, 200, defaultVarbinds())
		if err != nil {
			t.Fatalf("buildSNMPv3TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
	})

	t.Run("v3 auth priv SHA256/AES256C", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:       "sha256user",
			AuthProtocol:   g.SHA256,
			AuthPassphrase: "longenoughpassphrase256",
			PrivProtocol:   g.AES256C,
			PrivPassphrase: "longenoughprivphrase256",
			EngineID:       "test-engine-256",
		}
		data, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 300, defaultVarbinds())
		if err != nil {
			t.Fatalf("buildSNMPv3TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
	})

	t.Run("v3 missing username", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			AuthProtocol: g.NoAuth,
		}
		_, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 0, nil)
		if err == nil {
			t.Fatal("expected error for missing username")
		}
	})

	t.Run("v3 missing trap OID", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName: "testuser",
		}
		_, err := buildSNMPv3TrapPDU(secParams, "", 0, nil)
		if err == nil {
			t.Fatal("expected error for missing trap OID")
		}
	})

	t.Run("v3 default engine ID", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:     "testuser",
			AuthProtocol: g.NoAuth,
			PrivProtocol: g.NoPriv,
		}
		data, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 100, nil)
		if err != nil {
			t.Fatalf("buildSNMPv3TrapPDU() error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("returned empty bytes")
		}
	})

	t.Run("v3 noauth+priv rejected", func(t *testing.T) {
		secParams := SNMPv3SecurityParams{
			UserName:       "testuser",
			AuthProtocol:   g.NoAuth,
			PrivProtocol:   g.AES,
			PrivPassphrase: "privatekey12345",
			EngineID:       "test-engine",
		}
		_, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 100, defaultVarbinds())
		if err == nil {
			t.Fatal("expected error for NoAuth + Priv combination")
		}
		if !strings.Contains(err.Error(), "privacy requires authentication") {
			t.Errorf("expected privacy-requires-auth error, got: %v", err)
		}
	})
}

func TestBuildSNMPv3TrapPDU_ZeroTimestamp(t *testing.T) {
	secParams := SNMPv3SecurityParams{
		UserName:     "testuser",
		AuthProtocol: g.NoAuth,
		PrivProtocol: g.NoPriv,
		EngineID:     "test-engine",
	}
	data, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 0, defaultVarbinds())
	if err != nil {
		t.Fatalf("buildSNMPv3TrapPDU() with zero timestamp error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("returned empty bytes")
	}
	if data[0] != 0x30 {
		t.Errorf("expected SEQUENCE tag 0x30, got 0x%02X", data[0])
	}
}

func TestBuildSNMPv3TrapPDU_InitSecurityKeysError(t *testing.T) {
	origFn := initSecurityKeysFn
	defer func() { initSecurityKeysFn = origFn }()

	initSecurityKeysFn = func(_ *g.UsmSecurityParameters) error {
		return fmt.Errorf("injected key init failure")
	}

	secParams := SNMPv3SecurityParams{
		UserName:     "testuser",
		AuthProtocol: g.NoAuth,
		PrivProtocol: g.NoPriv,
		EngineID:     "test-engine",
	}
	_, err := buildSNMPv3TrapPDU(secParams, OIDColdStart, 100, nil)
	if err == nil {
		t.Fatal("expected error from injected InitSecurityKeys failure")
	}
	if !strings.Contains(err.Error(), "failed to init SNMPv3 security keys") {
		t.Errorf("expected security keys error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "injected key init failure") {
		t.Errorf("expected injected error message, got: %v", err)
	}
}

func TestSnmpVarbindsToPDUs(t *testing.T) {
	varbinds := []SNMPVarbind{
		{OID: "1.3.6.1.2.1.1.1.0", Type: g.OctetString, Value: "test"},
		{OID: "1.3.6.1.2.1.1.3.0", Type: g.TimeTicks, Value: uint32(12345)},
		{OID: "1.3.6.1.2.1.1.5.0", Type: g.Integer, Value: 42},
	}

	pdus := snmpVarbindsToPDUs(varbinds)

	if len(pdus) != 3 {
		t.Fatalf("expected 3 PDUs, got %d", len(pdus))
	}

	if pdus[0].Name != "1.3.6.1.2.1.1.1.0" {
		t.Errorf("expected OID 1.3.6.1.2.1.1.1.0, got %s", pdus[0].Name)
	}
	if pdus[0].Type != g.OctetString {
		t.Errorf("expected OctetString type, got %v", pdus[0].Type)
	}
}

func TestDefaultVarbinds(t *testing.T) {
	varbinds := defaultVarbinds()
	if len(varbinds) != 2 {
		t.Fatalf("expected 2 default varbinds, got %d", len(varbinds))
	}
	if varbinds[0].OID != OIDSysDescr {
		t.Errorf("expected first varbind OID %s, got %s", OIDSysDescr, varbinds[0].OID)
	}
	if varbinds[1].OID != OIDSysName {
		t.Errorf("expected second varbind OID %s, got %s", OIDSysName, varbinds[1].OID)
	}
}

func BenchmarkBuildSNMPv1TrapPDU(b *testing.B) {
	varbinds := defaultVarbinds()
	for i := 0; i < b.N; i++ {
		_, _ = buildSNMPv1TrapPDU("public", OIDEnterprise, "10.0.0.1", 6, 1, 12345, varbinds)
	}
}

func BenchmarkBuildSNMPv2cTrapPDU(b *testing.B) {
	varbinds := defaultVarbinds()
	for i := 0; i < b.N; i++ {
		_, _ = buildSNMPv2cTrapPDU("public", OIDColdStart, 12345, varbinds)
	}
}

func BenchmarkBuildSNMPv3TrapPDU_NoAuth(b *testing.B) {
	secParams := SNMPv3SecurityParams{
		UserName:     "testuser",
		AuthProtocol: g.NoAuth,
		PrivProtocol: g.NoPriv,
		EngineID:     "bench-engine",
	}
	varbinds := defaultVarbinds()
	for i := 0; i < b.N; i++ {
		_, _ = buildSNMPv3TrapPDU(secParams, OIDColdStart, 12345, varbinds)
	}
}
