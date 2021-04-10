package exporter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

// Neighbour type describing BGP Neighbour state
type Neighbour struct {
	connection net.Conn
	PeerIP     string
	PeerRID    net.IP
	Asn        uint32
	Asn32      bool
	HoldTime   uint16
	OptParams  []byte
	MyASN      uint32
	MyRID      net.IP
	routes     map[string]string
}

func (N *Neighbour) parceBGPOpenMsg(MessageBuf *[]byte) error {

	if uint8((*MessageBuf)[0]) != 4 {
		log.Error(N.PeerIP, " Incorrect BGP version")
		return errors.New("Incorrect BGP version")
	}

	N.Asn = uint32(binary.BigEndian.Uint16((*MessageBuf)[1:3]))
	N.HoldTime = binary.BigEndian.Uint16((*MessageBuf)[3:5])
	N.PeerRID = (*MessageBuf)[5:9]

	for _, Options := range parseTLV((*MessageBuf)[10:]) {
		if Options.Type == BGP_OPT_CAPABILITY {
			for _, Capability := range parseTLV(Options.Value) {

				switch Capability.Type {
				case BGP_OPT_CAP_ASN_32BIT:
					N.Asn32 = true
					N.Asn = binary.BigEndian.Uint32(Capability.Value)
					log.Debug(N.PeerIP, " Support ASN 32bit")
					break
				case BGP_OPT_CAP_ROUTE_REFRESH:
					log.Debug(N.PeerIP, " Support Route refresh")
					break
				case BGP_OPT_CAP_MP:
					log.Debug(N.PeerIP, " Support Multi Protocol ", Capability.Value)
					break
				}
			}
		} else {
			log.Info(N.PeerIP, " Received unsupported OPEN option")
		}
	}

	log.Debug(N.PeerIP, " Open received")
	return nil
}

func (N *Neighbour) sendBGPOpenMsg() error {
	buf := make([]byte, 4)

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

	Capabilities := append(BGP_CAP_MP_IPv4_UNICAST, BGP_CAP_32_BIT_ASN...)
	binary.BigEndian.PutUint32(buf, N.MyASN)
	Capabilities = append(Capabilities, buf...)
	//Capabilities = append(Capabilities, BGP_CAP_MP_IPv6_UNICAST...)

	OpenMsg.OptParams = append([]byte{BGP_OPT_CAPABILITY, uint8(len(Capabilities))}, Capabilities...)

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
	Buffer.Write(OpenMsg.BGPIdentifier)
	binary.Write(Buffer, binary.BigEndian, &OpenMsg.OptLen)
	Buffer.Write(OpenMsg.OptParams)

	N.connection.Write(Buffer.Bytes())

	return nil
}

func (N *Neighbour) sendBGPKeepaliveMsg() error {
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

func aspathtostr(aspath []uint32) string {
	aspathstring := []string{}
	for _, asn := range aspath {
		aspathstring = append(aspathstring, strconv.FormatUint(uint64(asn), 10))
	}
	return strings.Join(aspathstring, "_")
}

func ipv4ttostr(route Route) string {
	routelen := 4
	ipaddrstr := []string{}
	for i := 0; i < routelen; i++ {
		if i < len(route.Prefix) {
			ipaddrstr = append(ipaddrstr, strconv.FormatUint(uint64(route.Prefix[i]), 10))
		} else {
			ipaddrstr = append(ipaddrstr, "0")
		}
	}
	return strings.Join(ipaddrstr, ".") + "/" + strconv.FormatUint(uint64(route.PrefixLen), 10)
}

func (N *Neighbour) handleBGPUpdateMsg(UpdateBuf *[]byte) {
	UpdateMsg := parceBGPUpdateMsg(UpdateBuf)
	aspath := []uint32{}
	log.Debug("UpdateMsg: ", UpdateMsg)
	//Rarce Path attributes
	if len(UpdateMsg.PathArrtibutes) > 0 {
		if pa, ok := UpdateMsg.PathArrtibutes[BGP_PA_ASPATH]; ok {
			if len(pa.Value) > 0 {
				for i := 0; i < int(pa.Value[1]); i++ {
					if N.Asn32 {
						aspath = append(aspath, (binary.BigEndian.Uint32(pa.Value[(i*4)+2 : (i*4)+6])))
					} else {
						aspath = append(aspath, uint32((binary.BigEndian.Uint16(pa.Value[(i*2)+2 : (i*2)+4]))))
					}
				}
			} else {
				log.Error(N.PeerIP, " ASPATH length is 0")
			}
		}
	}

	//Add routes to route table
	if len(UpdateMsg.NLRI) > 0 {
		for _, route := range UpdateMsg.NLRI { //need SWAP LOOPS for big routing tables
			route.AsPath = append(route.AsPath, aspath...)
			if existaspath, ok := N.routes[ipv4ttostr(route)]; ok {
				routes.WithLabelValues(N.PeerIP, ipv4ttostr(route), existaspath).Dec()
				neighbourRoutes.WithLabelValues(N.PeerIP).Dec()
			}
			N.routes[ipv4ttostr(route)] = aspathtostr(route.AsPath)
			routes.WithLabelValues(N.PeerIP, ipv4ttostr(route), aspathtostr(route.AsPath)).Inc()
			routeChange.WithLabelValues(N.PeerIP, ipv4ttostr(route), aspathtostr(route.AsPath)).Inc()
			neighbourRoutes.WithLabelValues(N.PeerIP).Inc()
		}
	}
	//Delete Withdrawn Routes from route table
	if len(UpdateMsg.WithdrawnRoutes) > 0 {
		for _, route := range UpdateMsg.WithdrawnRoutes {
			if existaspath, ok := N.routes[ipv4ttostr(route)]; ok {
				routes.WithLabelValues(N.PeerIP, ipv4ttostr(route), existaspath).Dec()
				routeChange.WithLabelValues(N.PeerIP, ipv4ttostr(route), existaspath).Inc()
			}
			delete(N.routes, ipv4ttostr(route))
			neighbourRoutes.WithLabelValues(N.PeerIP).Dec()
		}
	}
}

func parceBGPUpdateMsg(UpdateBuf *[]byte) BGPUpdateMsg {
	UpdateMsg := BGPUpdateMsg{}

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
			parcedroute.Prefix = buff[index+1 : index+int(parcedroute.PrefixLen/8)+2]
			index = index + int(parcedroute.PrefixLen)/8 + 2
		} else {
			parcedroute.Prefix = buff[index+1 : index+int(parcedroute.PrefixLen/8)+1]
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

// HandlePeer handle incoming connections.
func HandlePeer(conn net.Conn, cfg *Config) {
	PeerIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	Peer := Neighbour{
		connection: conn,
		PeerIP:     PeerIP,
		Asn32:      false,
		MyASN:      uint32(cfg.Asn),
		MyRID:      cfg.Rid.To4(),
		routes:     make(map[string]string),
	}
	var MessageBuf []byte
	Header := BGPHeader{}

	totalConnections.Inc()
	aliveConnections.Inc()
	log.Info(Peer.PeerIP, " New connection")
loop:
	for {

		err := binary.Read(conn, binary.BigEndian, &Header)
		if err != nil {
			log.Error(Peer.PeerIP, " Connection read failed:", err)
			break
		}
		//fmt.Println(Header)
		if !reflect.DeepEqual(Header.Padding, BGP_HEADER_PADDING) {
			log.Error(Peer.PeerIP, " Not BGP Packet")
			break
		}
		if Header.Length > 0 {
			MessageBuf, err = readBytes(conn, (int(Header.Length) - binary.Size(Header)))
			if err != nil {
				log.Error(Peer.PeerIP, " Message body read failed:", err)
				break
			}
		}

		switch Header.Type {
		case BGP_MSG_OPEN:
			err := Peer.parceBGPOpenMsg(&MessageBuf)
			if err != nil {
				log.Error(Peer.PeerIP, " OPEN failed:", err)
				break loop
			}

			err = Peer.sendBGPOpenMsg()
			if err != nil {
				log.Error(Peer.PeerIP, " Open send failed:", err)
				break loop
			}
		case BGP_MSG_UPDATE:
			log.Debug(Peer.PeerIP, " Update received")
			Peer.handleBGPUpdateMsg(&MessageBuf)

		case BGP_MSG_NOTIFICATION:
			log.Error(Peer.PeerIP, " Notification received: ", MessageBuf)
			break loop
		case BGP_MSG_KEEPALIVE:
			log.Debug(Peer.PeerIP, " Keepalive received")
			err := Peer.sendBGPKeepaliveMsg()
			if err != nil {
				log.Print(Peer.PeerIP, " Keepalive send failed:", err)
				break loop
			}
		case BGP_MSG_REFRESH:
			//Unsupported
			log.Debug(Peer.PeerIP, "Refresh received")
			fmt.Println(MessageBuf)
		}

		MessageBuf = nil

	}
	log.Info(Peer.PeerIP, " Close connection")
	conn.Close()
	aliveConnections.Dec()
	for route, aspath := range Peer.routes {
		if cfg.DeleteOnDisconnect {
			routeChange.DeleteLabelValues(Peer.PeerIP, route, aspath)
			routes.DeleteLabelValues(Peer.PeerIP, route, aspath)
			neighbourRoutes.DeleteLabelValues(Peer.PeerIP)
		} else {
			routeChange.WithLabelValues(Peer.PeerIP, route, aspath).Inc()
			routes.WithLabelValues(Peer.PeerIP, route, aspath).Dec()
			neighbourRoutes.WithLabelValues(Peer.PeerIP).Set(0)
		}
	}
}
