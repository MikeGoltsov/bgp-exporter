package main

import "net"

var (
	BGP_HEADER_PADDING      = [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	BGP_CAP_MP_IPv4_UNICAST = []byte{0x01, 0x04, 0x00, 0x01, 0x00, 0x01}
	BGP_CAP_MP_IPv6_UNICAST = []byte{0x01, 0x04, 0x00, 0x02, 0x00, 0x01}
	BGP_CAP_32_BIT_ASN      = []byte{0x41, 0x04}
)

const (
	BGP_HEADER_LENGTH      = 19
	BGP_MAX_MESSAGE_LENGTH = 4096
	BGP_OPEN_FIX_LENGTH    = 10

	BGP_MSG_OPEN        = 1
	BGP_MSG_UPDATE      = 2
	BGP_MSG_NOTIFICATON = 3
	BGP_MSG_KEEPALIVE   = 4
	BGP_MSG_REFRESH     = 5

	BGP_AS_TRANS = 23456

	BGP_OPT_CAPABILITY        = 0x02
	BGP_OPT_CAP_ASN_32BIT     = 0x41
	BGP_OPT_CAP_ROUTE_REFRESH = 0x02
	BGP_OPT_CAP_MP            = 0x01

	BGP_PA_ASPATH = 2
)

type BGPHeader struct {
	Padding [16]byte
	Length  uint16
	Type    uint8
}

type BGPOpenMsg struct {
	Version       uint8
	Asn           uint16
	HoldTime      uint16
	BGPIdentifier net.IP
	OptLen        uint8
	OptParams     []byte
}

type BGPUpdateMsg struct {
	WithdrawnRoutesLen uint16
	WithdrawnRoutes    []Route
	PathArrtibutesLen  uint16
	PathArrtibutes     map[uint8]PathAttr
	NLRI               []Route
}

type TLV struct {
	Type   uint8
	Length uint8
	Value  []byte
}

type Route struct {
	PrefixLen uint8
	Prefix    []byte
	AsPath    []uint32
}

type PathAttr struct {
	flags uint8
	Value []byte
}
