package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

const msgHi = byte(1)
const msgOwnIP = byte(2)
const msgPeerIP = byte(3)

var first net.Addr
var second net.Addr

func makeReply(id byte, addr net.Addr) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, []byte(addr.String()))
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
	log.Println("Listening on", conn.LocalAddr())

	conn.SetReadBuffer(1048576)

	for {
		buffer := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Println(err.Error())
			return
		}

		data := buffer[:n]

		log.Println("Received", data, addr)

		r := bytes.NewReader(data)

		var code byte
		binary.Read(r, binary.LittleEndian, &code)

		if code == msgHi {
			if first == nil {
				first = addr
				_, err := conn.WriteTo(makeReply(msgOwnIP, first), first)
				if err != nil {
					log.Println(err.Error())
					return
				}
			} else {
				second = addr
				conn.WriteTo(makeReply(msgOwnIP, second), second)
				conn.WriteTo(makeReply(msgPeerIP, second), first)
				conn.WriteTo(makeReply(msgPeerIP, first), second)
				first = nil
				second = nil
			}
		}
	}
}
