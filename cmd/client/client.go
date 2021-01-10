package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
)

// Network code indicating the type of message.
const (
	MsgCodeJoin      = byte(1) // Create or join a netplay room
	MsgCodeOwnIP     = byte(2) // Get to know your own external IP as well as your player index
	MsgCodePeerIP    = byte(3) // Get the IP of your peer, along with its player index
	MsgCodeHandshake = byte(4) // For both peer to contact each others
)

func makeJoinPacket(crc uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, MsgCodeJoin)
	binary.Write(buf, binary.LittleEndian, crc)
	return buf.Bytes()
}

func makeHandshakePacket() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, MsgCodeHandshake)
	return buf.Bytes()
}

func receiveReply(conn *net.UDPConn) (int, string, error) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return 0, "", err
	}
	data := buffer[:n]

	log.Println("Received", data)

	r := bytes.NewReader(data)

	var code byte

	binary.Read(r, binary.LittleEndian, &code)
	if code == MsgCodeOwnIP || code == MsgCodePeerIP {
		var playerID byte
		binary.Read(r, binary.LittleEndian, &playerID)
		addr := data[2:]
		return int(playerID), string(addr), nil
	}

	return 0, "", nil
}

func main() {
	port, _ := strconv.ParseInt(os.Args[2], 10, 64)
	rdv, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(os.Args[1]),
		Port: int(port),
	})
	if err != nil {
		log.Println(err.Error())
		return
	}

	rdv.SetReadBuffer(1048576)

	_, err = rdv.Write(makeJoinPacket(42))
	if err != nil {
		log.Println(err.Error())
		return
	}

	myID, my, _ := receiveReply(rdv)
	log.Println("I am", my, ", Player #", myID)

	myIP, myPortStr, err := net.SplitHostPort(my)
	if err != nil {
		log.Println(err.Error())
		return
	}
	myPort, _ := strconv.ParseInt(myPortStr, 10, 64)

	peerID, peer, _ := receiveReply(rdv)
	log.Println("I see", peer, ", Player #", peerID)

	peerIP, peerPortStr, err := net.SplitHostPort(peer)
	if err != nil {
		log.Println(err.Error())
		return
	}
	peerPort, _ := strconv.ParseInt(peerPortStr, 10, 64)
	peerAddr := &net.UDPAddr{
		IP:   net.ParseIP(peerIP),
		Port: int(peerPort),
	}

	rdv.Close()

	p2p, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(myIP),
		Port: int(myPort),
	})
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("Listening on", p2p.LocalAddr())

	p2p.SetReadBuffer(1048576)

	log.Println("Sending hello")
	_, err = p2p.WriteTo(makeHandshakePacket(), peerAddr)
	if err != nil {
		log.Println(err.Error())
		return
	}

	for {
		_, msg, _ := receiveReply(p2p)
		log.Println(msg)
		return
	}
}
