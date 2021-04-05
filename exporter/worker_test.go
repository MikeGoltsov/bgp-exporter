package exporter

import (
	"testing"
)

func TestParceTLV(t *testing.T) {
	v := parseTLV([]byte{0x01, 0x02, 0x05, 0x01})
	if v[0].Type != 1 || len(v[0].Value) != 2 {
		t.Error("Parced incrrect, len:", len(v[0].Value), v)
	}
}

func TestIpv4ttostr(t *testing.T) {
	testroute := Route{
		Prefix:    []byte{0x01, 0x01, 0x01, 0x01},
		PrefixLen: 24,
	}
	string := ipv4ttostr(testroute)
	if string != "1.1.1.1/24" {
		t.Error("Parced incrrect", string)
	}
}
