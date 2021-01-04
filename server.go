package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

const msgHi = 1
const msgOwnIP = 2
const msgPeerIP = 3

type peer struct {
	net.Addr
}

var first *peer
var second *peer

func makeReply(id byte, addr net.Addr) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, addr.String())
	return buf.Bytes()
}

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 1234,
	})
	if err != nil {
		log.Println(err.Error())
		return
	}

	conn.SetReadBuffer(1048576)

	for {
		buffer := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Println(err.Error())
			return
		}

		data := buffer[:n]

		r := bytes.NewReader(data)

		var code byte
		binary.Read(r, binary.LittleEndian, &code)

		if code == msgHi {
			if first == nil {
				first = &peer{addr}
				conn.WriteTo(makeReply(msgOwnIP, first), first)
			} else {
				second = &peer{addr}
				conn.WriteTo(makeReply(msgOwnIP, second), second)
				conn.WriteTo(makeReply(msgPeerIP, second), first)
				conn.WriteTo(makeReply(msgPeerIP, first), second)
				first = nil
				second = nil
			}
		}
	}
}
