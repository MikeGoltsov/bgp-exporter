package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

type Neighbour struct {
	PeerIP  string
	Asn     uint32
	Options []TLV
}

func ParceBGPOpenMsg(OpenBuf []byte, cfg config) BGPOpenMsg {
	OpenMsg := BGPOpenMsg{}

	OpenMsg.Version = uint8(OpenBuf[0])
	OpenMsg.Asn = binary.BigEndian.Uint16(OpenBuf[1:3])
	OpenMsg.HoldTime = binary.BigEndian.Uint16(OpenBuf[3:5])
	OpenMsg.BGPIdentifier = OpenBuf[5:9]
	OpenMsg.OptLen = uint8(OpenBuf[9])
	OpenMsg.OptParams = OpenBuf[10:]
	log.Info("Open recieved: ", OpenMsg.BGPIdentifier)

	return OpenMsg
}

func SendBGPOpenMsg(conn net.Conn, cfg config) error {
	Header := BGPHeader{
		Padding: BGP_HEADER_PADDING,
		Type:    BGP_MSG_OPEN}
	OpenMsg := BGPOpenMsg{
		Version:       4,
		Asn:           uint16(cfg.Asn),
		HoldTime:      180,
		BGPIdentifier: cfg.rid.To4(),
	}

	OpenMsg.OptLen = uint8(len(OpenMsg.OptParams))
	Header.Length = uint16(binary.Size(Header) + BGP_OPEN_FIX_LENGTH + len(OpenMsg.OptParams))

	Buffer := new(bytes.Buffer)
	err := binary.Write(Buffer, binary.BigEndian, &Header)
	if err != nil {
		log.Error("Connection header write failed:", err)
		return err
	}
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.Version)
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.Asn)
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.HoldTime)
	Buffer.Write(OpenMsg.BGPIdentifier)
	err = binary.Write(Buffer, binary.BigEndian, &OpenMsg.OptLen)
	if err != nil {
		log.Error("Connection open msg write failed:", err)
		return err
	}
	Buffer.Write(OpenMsg.OptParams)

	conn.Write(Buffer.Bytes())
	return nil
}

func SendBGPKeepaliveMsg(conn net.Conn) error {
	Header := BGPHeader{
		Padding: BGP_HEADER_PADDING,
		Length:  BGP_HEADER_LENGTH,
		Type:    BGP_MSG_KEEPALIVE}
	err := binary.Write(conn, binary.BigEndian, &Header)
	if err != nil {
		log.Error("Connection write failed:", err)
		return err
	}
	return nil
}

func ParceBGPUpdateMsg(UpdateBuf []byte) BGPUpdateMsg {
	UpdateMsg := BGPUpdateMsg{}
	fmt.Println("UpdateBuff: ", UpdateBuf)

	//Parce WithdrawnRoutes
	UpdateMsg.WithdrawnRoutesLen = binary.BigEndian.Uint16(UpdateBuf[0:2])
	if UpdateMsg.WithdrawnRoutesLen == 0 {
		UpdateMsg.WithdrawnRoutes = []byte{}
	} else {
		UpdateMsg.WithdrawnRoutes = parceRoute(UpdateBuf[2:UpdateMsg.WithdrawnRoutesLen])
	}
	index := int(unsafe.Sizeof(UpdateMsg.WithdrawnRoutesLen)) + int(UpdateMsg.WithdrawnRoutesLen)

	//Parce PathArrtibutes
	UpdateMsg.PathArrtibutesLen = binary.BigEndian.Uint16(UpdateBuf[index : index+2])
	if UpdateMsg.PathArrtibutesLen == 0 {
		UpdateMsg.PathArrtibutes = []byte{}
	} else {
		UpdateMsg.PathArrtibutes = UpdateBuf[index+2 : index+int(UpdateMsg.PathArrtibutesLen)]
	}
	index = index + int(unsafe.Sizeof(UpdateMsg.PathArrtibutesLen)) + int(UpdateMsg.PathArrtibutesLen)

	//Parce NLRI
	if len(UpdateBuf[index:]) > 0 {
		UpdateMsg.NLRI = parceRoute(UpdateBuf[index:])
	}
	fmt.Println("UpdateMsg: ", UpdateMsg)
	return UpdateMsg
}

func parceRoute(buff []byte) []Route {
	len := len(buff)
	fmt.Println("buff route", buff)

	var parcedroute Route
	r := make([]Route, 0)
	for index := 0; ; {
		parcedroute.PrefixLen = uint8(buff[index])
		fmt.Println("route len ", parcedroute.PrefixLen)
		if uint8(buff[index])%8 != 0 {
			parcedroute.Value = buff[index+1 : index+int(parcedroute.PrefixLen/8)+2]
			index = index + int(parcedroute.PrefixLen)/8 + 2
		} else {
			parcedroute.Value = buff[index+1 : index+int(parcedroute.PrefixLen/8)+1]
			index = index + int(parcedroute.PrefixLen)/8 + 1
		}
		r = append(r, parcedroute)

		//	fmt.Println("parce route", r, index, len)
		if index >= len {
			break
		}
	}
	return r
}

func parseTLV(buff []byte) []TLV {
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

func readBytes(conn net.Conn, length int) ([]byte, error) {
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
		PeerIP: PeerIP,
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
			OpenMsg := ParceBGPOpenMsg(MessageBuf, cfg)
			//fmt.Println(BGPPeer.PeerIP, "OpenMSG: ", OpenMsg)

			for _, Options := range parseTLV(OpenMsg.OptParams) {
				if Options.Type == 2 {
					BGPPeer.Options = parseTLV(Options.Value)
					log.Debug(BGPPeer.PeerIP, " Open Options list: ", BGPPeer.Options)
				} else {
					log.Error(BGPPeer.PeerIP, " Cant parse Capabilities option")
				}
			}
			err = SendBGPOpenMsg(conn, cfg)
			if err != nil {
				log.Error(BGPPeer.PeerIP, " Open send failed:", err)
				break
			}
		case BGP_MSG_UPDATE:
			log.Debug(BGPPeer.PeerIP, " Update recieved")
			_ = ParceBGPUpdateMsg(MessageBuf)

		case BGP_MSG_NOTIFICATON:
			log.Error(BGPPeer.PeerIP, " Notification recieved: ", MessageBuf)
			break loop
		case BGP_MSG_KEEPALIVE:
			log.Debug(BGPPeer.PeerIP, " Keepalive recieved")
			err := SendBGPKeepaliveMsg(conn)
			if err != nil {
				log.Print(BGPPeer.PeerIP, " Keepalive send failed:", err)
				break
			}
		case BGP_MSG_REFRESH:
			log.Debug(BGPPeer.PeerIP, "Refresh recieved")
			fmt.Println(MessageBuf)
		}

		// Send a response back to person contacting us.
		//conn.Write([]byte("Message received."))
		// Close the connection when you're done with it.
		MessageBuf = nil

	}
	log.Info(BGPPeer.PeerIP, " Close connection")
	conn.Close()
	aliveConnections.Dec()
}
