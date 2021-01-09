package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
)

const msgHi = byte(1)
const msgOwnIP = byte(2)
const msgPeerIP = byte(3)

func makeHi() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, msgHi)
	return buf.Bytes()
}

func receiveReply(conn *net.UDPConn) string {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	data := buffer[:n]

	log.Println("Received", data)

	r := bytes.NewReader(data)

	var code byte
	var playerID byte
	var addr []byte
	binary.Read(r, binary.LittleEndian, &code)
	if code == msgOwnIP || code == msgPeerIP {
		binary.Read(r, binary.LittleEndian, &playerID)
		addr = data[2:]
	}

	return string(addr)
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

	_, err = rdv.Write(makeHi())
	if err != nil {
		log.Println(err.Error())
		return
	}

	my := receiveReply(rdv)
	log.Println("I am", my)

	myIP, myPortStr, err := net.SplitHostPort(my)
	if err != nil {
		log.Println(err.Error())
		return
	}
	myPort, _ := strconv.ParseInt(myPortStr, 10, 64)

	peer := receiveReply(rdv)
	log.Println("I see", peer)

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
	_, err = p2p.WriteTo(makeHi(), peerAddr)
	if err != nil {
		log.Println(err.Error())
		return
	}

	for {
		msg := receiveReply(p2p)
		log.Println(msg)
		return
	}
}
