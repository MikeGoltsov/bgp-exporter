package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

type Neighbour struct {
	connection net.Conn
	PeerIP     string
	PeerRID    net.IP
	Asn        uint32
	Asn32      bool
	Options    []TLV
	HoldTime   uint16
	OptParams  []byte
	MyASN      uint32
	MyRID      net.IP
}

func (N *Neighbour) ParceBGPOpenMsg(MessageBuf *[]byte) error {

	if uint8((*MessageBuf)[0]) != 4 {
		log.Error(N.PeerIP, " Incorrect BGP version")
		return errors.New("Incorrect BGP version")
	}

	N.Asn = uint32(binary.BigEndian.Uint16((*MessageBuf)[1:3]))
	N.HoldTime = binary.BigEndian.Uint16((*MessageBuf)[3:5])
	N.PeerRID = (*MessageBuf)[5:9]

	for _, Options := range parseTLV((*MessageBuf)[10:]) {
		if Options.Type == BGP_OPT_CAPABILITY {
			N.Options = parseTLV(Options.Value)
			log.Debug(N.PeerIP, " Open Options list: ", N.Options)
		} else {
			log.Error(N.PeerIP, " Cant parse Capabilities option")
		}
	}

	log.Info(N.PeerIP, " Open recieved")
	return nil
}

func (N *Neighbour) SendBGPOpenMsg() error {
	Header := BGPHeader{
		Padding: BGP_HEADER_PADDING,
		Type:    BGP_MSG_OPEN}
	OpenMsg := BGPOpenMsg{
		Version:       4,
		HoldTime:      180,
		BGPIdentifier: N.MyRID,
	}

	if N.MyASN > (1<<16)-1 { //Check 32 bit ASN
		OpenMsg.Asn = BGP_AS_TRANS
	} else {
		OpenMsg.Asn = uint16(N.MyASN)
	}

	Capabilities := append(BGP_CAP_MP_IPv4_UNICAST)

	Capabilities = append(Capabilities, BGP_CAP_32_BIT_ASN...)
	ansbuf := make([]byte, 4)
	binary.BigEndian.PutUint32(ansbuf, N.MyASN)
	Capabilities = append(Capabilities, ansbuf...)

	OpenMsg.OptParams = append([]byte{0x02, uint8(len(Capabilities))}, Capabilities...)

	OpenMsg.OptLen = uint8(len(OpenMsg.OptParams))
	Header.Length = uint16(binary.Size(Header) + BGP_OPEN_FIX_LENGTH + len(OpenMsg.OptParams))

	Buffer := new(bytes.Buffer)
	err := binary.Write(Buffer, binary.BigEndian, &Header)
	if err != nil {
		log.Error("Connection OPEN header write failed:", err)
		return err
	}
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.Version)
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.Asn)
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.HoldTime)
	Buffer.Write(N.MyRID)
	err = binary.Write(Buffer, binary.BigEndian, &OpenMsg.OptLen)
	if err != nil {
		log.Error("Connection OPEN msg write failed:", err)
		return err
	}
	Buffer.Write(OpenMsg.OptParams)

	N.connection.Write(Buffer.Bytes())

	return nil
}

func (N *Neighbour) SendBGPKeepaliveMsg() error {
	Header := BGPHeader{
		Padding: BGP_HEADER_PADDING,
		Length:  BGP_HEADER_LENGTH,
		Type:    BGP_MSG_KEEPALIVE}
	err := binary.Write(N.connection, binary.BigEndian, &Header)
	if err != nil {
		log.Error("Connection write failed:", err)
		return err
	}
	return nil
}

func ParceBGPUpdateMsg(UpdateBuf *[]byte) BGPUpdateMsg {
	UpdateMsg := BGPUpdateMsg{}
	//fmt.Println("UpdateBuff: ", *UpdateBuf)

	//Parce WithdrawnRoutes
	UpdateMsg.WithdrawnRoutesLen = binary.BigEndian.Uint16((*UpdateBuf)[0:2])
	if UpdateMsg.WithdrawnRoutesLen == 0 {
		UpdateMsg.WithdrawnRoutes = []Route{}
	} else {
		UpdateMsg.WithdrawnRoutes = parceRoute((*UpdateBuf)[2:UpdateMsg.WithdrawnRoutesLen])
	}
	index := int(unsafe.Sizeof(UpdateMsg.WithdrawnRoutesLen)) + int(UpdateMsg.WithdrawnRoutesLen)

	//Parce PathArrtibutes
	UpdateMsg.PathArrtibutesLen = binary.BigEndian.Uint16((*UpdateBuf)[index : index+2])
	if UpdateMsg.PathArrtibutesLen == 0 {
		UpdateMsg.PathArrtibutes = map[uint8]PathAttr{}
	} else {
		UpdateMsg.PathArrtibutes = parcePathAttr((*UpdateBuf)[index+2 : index+int(UpdateMsg.PathArrtibutesLen)])
	}
	index = index + int(unsafe.Sizeof(UpdateMsg.PathArrtibutesLen)) + int(UpdateMsg.PathArrtibutesLen)

	//Parce NLRI
	if len((*UpdateBuf)[index:]) > 0 {
		UpdateMsg.NLRI = parceRoute((*UpdateBuf)[index:])
	}
	return UpdateMsg
}

func parceRoute(buff []byte) []Route {
	len := len(buff)
	var parcedroute Route
	r := make([]Route, 0)
	for index := 0; ; {
		parcedroute.PrefixLen = uint8(buff[index])
		if uint8(buff[index])%8 != 0 {
			parcedroute.Value = buff[index+1 : index+int(parcedroute.PrefixLen/8)+2]
			index = index + int(parcedroute.PrefixLen)/8 + 2
		} else {
			parcedroute.Value = buff[index+1 : index+int(parcedroute.PrefixLen/8)+1]
			index = index + int(parcedroute.PrefixLen)/8 + 1
		}
		r = append(r, parcedroute)
		if index >= len {
			break
		}
	}
	return r
}

func parcePathAttr(buff []byte) map[uint8]PathAttr {
	//fmt.Println("PAbuff: ", buff)
	len := len(buff)
	var attrLen int
	parcedPA := PathAttr{}
	a := make(map[uint8]PathAttr, 0)
	for index := 0; ; {
		parcedPA.flags = buff[index]
		if (buff[index] & 16) > 0 { //check if 5 bit is set (extended length)
			attrLen = int(binary.BigEndian.Uint16(buff[index+2:index+4])) + 4
			parcedPA.Value = buff[index+4 : index+attrLen]
		} else {
			attrLen = int(buff[index+2]) + 3
			parcedPA.Value = buff[index+3 : index+attrLen]
		}
		a[buff[index+1]] = parcedPA
		//fmt.Println("PAmap: ", a, index, len)
		index = index + attrLen
		if index >= len {
			break
		}
	}
	return a
}

func parseTLV(buff []byte) []TLV { // Parce TLV to struct
	len := len(buff)
	//fmt.Println("len:", len)
	var parcedtlv TLV
	m := make([]TLV, 0)
	for index := 0; ; {
		parcedtlv.Type = buff[index]
		parcedtlv.Length = buff[index+1]
		if parcedtlv.Length == 0 {
			parcedtlv.Value = []byte{}
		} else {
			parcedtlv.Value = buff[index+2 : index+int(buff[index+1])+2]
		}
		index = index + int(buff[index+1]) + 2
		m = append(m, parcedtlv)
		parcedtlv = TLV{}
		//fmt.Println(m, index, len)
		if index >= len {
			break
		}
	}
	return m
}

func readBytes(conn net.Conn, length int) ([]byte, error) { //Read bytes from net socket
	buf := make([]byte, length)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Handles incoming requests.
func handlePeer(conn net.Conn, cfg config) {
	PeerIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	BGPPeer := Neighbour{
		connection: conn,
		PeerIP:     PeerIP,
		MyASN:      uint32(cfg.Asn),
		MyRID:      cfg.rid.To4(),
	}
	var MessageBuf []byte
	Header := BGPHeader{}

	totalConnections.Inc()
	aliveConnections.Inc()
	log.Info(BGPPeer.PeerIP, " New connection")
loop:
	for {

		err := binary.Read(conn, binary.BigEndian, &Header)
		if err != nil {
			log.Error(BGPPeer.PeerIP, " Connection read failed:", err)
			break
		}
		//fmt.Println(Header)
		if !reflect.DeepEqual(Header.Padding, BGP_HEADER_PADDING) {
			log.Error(BGPPeer.PeerIP, " Not BGP Packet")
			break
		}
		if Header.Length > 0 {
			MessageBuf, err = readBytes(conn, (int(Header.Length) - binary.Size(Header)))
			if err != nil {
				log.Error(BGPPeer.PeerIP, " Message body read failed:", err)
				break
			}
		}

		switch Header.Type {
		case BGP_MSG_OPEN:
			err := BGPPeer.ParceBGPOpenMsg(&MessageBuf)
			if err != nil {
				log.Error(BGPPeer.PeerIP, " Ceonnetion failed:", err)
				break
			}

			fmt.Println("BGPPeer: ", BGPPeer)

			err = BGPPeer.SendBGPOpenMsg()
			if err != nil {
				log.Error(BGPPeer.PeerIP, " Open send failed:", err)
				break
			}
		case BGP_MSG_UPDATE:
			log.Debug(BGPPeer.PeerIP, " Update recieved")
			UpdateMsg := ParceBGPUpdateMsg(&MessageBuf)
			log.Debug("UpdateMsg: ", UpdateMsg)
		case BGP_MSG_NOTIFICATON:
			log.Error(BGPPeer.PeerIP, " Notification recieved: ", MessageBuf)
			break loop
		case BGP_MSG_KEEPALIVE:
			log.Debug(BGPPeer.PeerIP, " Keepalive recieved")
			err := BGPPeer.SendBGPKeepaliveMsg()
			if err != nil {
				log.Print(BGPPeer.PeerIP, " Keepalive send failed:", err)
				break
			}
		case BGP_MSG_REFRESH:
			//Unsupported
			log.Debug(BGPPeer.PeerIP, "Refresh recieved")
			fmt.Println(MessageBuf)
		}

		MessageBuf = nil

	}
	log.Info(BGPPeer.PeerIP, " Close connection")
	conn.Close()
	aliveConnections.Dec()
}
