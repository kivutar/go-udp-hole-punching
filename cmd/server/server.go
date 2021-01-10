package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

const (
	msgJoin = byte(1)
	msgIP   = byte(2)
)

var first net.Addr
var second net.Addr

func makeReply(id byte, playerID byte, addr net.Addr) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, playerID)
	binary.Write(buf, binary.LittleEndian, []byte(addr.String()))
	return buf.Bytes()
}

func receive(conn *net.UDPConn) error {
	buffer := make([]byte, 1024)
	n, addr, err := conn.ReadFrom(buffer)
	if err != nil {
		return err
	}

	data := buffer[:n]

	log.Println("Received", data, "from", addr)

	r := bytes.NewReader(data)

	var code byte
	binary.Read(r, binary.LittleEndian, &code)

	switch code {
	case msgJoin:
		if first == nil {
			first = addr
			_, err := conn.WriteTo(makeReply(msgIP, 0, first), first)
			if err != nil {
				return err
			}
		} else {
			second = addr
			conn.WriteTo(makeReply(msgIP, 1, second), second)
			conn.WriteTo(makeReply(msgIP, 1, second), first)
			conn.WriteTo(makeReply(msgIP, 0, first), second)
			first = nil
			second = nil
		}
	default:
		log.Println("Received unknown message")
	}

	return nil
}

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 1234,
	})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Listening on", conn.LocalAddr())

	conn.SetReadBuffer(1048576)

	for {
		err := receive(conn)
		if err != nil {
			log.Println(err)
		}
	}
}
