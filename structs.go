package main

import "net"

var (
	BGP_HEADER_PADDING = [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
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
	WithdrawnRoutes    []byte
	PathArrtibutesLen  uint16
	PathArrtibutes     []byte
	NLRI               []byte
}

type TLV struct {
	Type   uint8
	Length uint8
	Value  []byte
}
