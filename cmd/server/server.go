package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"
)

// Network code indicating the type of message.
const (
	MsgCodeJoin      = byte(1) // Create or join a netplay room
	MsgCodeOwnIP     = byte(2) // Get to know your own external IP as well as your player index
	MsgCodePeerIP    = byte(3) // Get the IP of your peer, along with its player index
	MsgCodeHandshake = byte(4) // For both peer to contact each others
)

// Room is a game room where 2 players connect
type Room struct {
	CRC       uint32
	Players   []net.Addr
	CreatedAt time.Time
}

// Rooms is the list of rooms
var Rooms []Room

func findRoom(crc uint32, addr net.Addr) *Room {
	for _, r := range Rooms {
		if r.CRC == crc &&
			len(r.Players) == 1 && r.Players[0] != addr &&
			r.CreatedAt.After(time.Now().Add(-time.Minute)) {
			return &r
		}
	}
	return nil
}

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
	case MsgCodeJoin:
		var crc uint32
		binary.Read(r, binary.LittleEndian, &crc)
		room := findRoom(crc, addr)

		if room != nil {
			room.Players = append(room.Players, addr)
			conn.WriteTo(makeReply(MsgCodeOwnIP, 1, room.Players[1]), room.Players[1])
			conn.WriteTo(makeReply(MsgCodePeerIP, 1, room.Players[1]), room.Players[0])
			conn.WriteTo(makeReply(MsgCodePeerIP, 0, room.Players[0]), room.Players[1])
			log.Println("Player", addr, "Joined room", *room)
		} else {
			room := Room{
				CRC:       crc,
				Players:   []net.Addr{addr},
				CreatedAt: time.Now(),
			}
			Rooms = append(Rooms, room)
			_, err := conn.WriteTo(makeReply(MsgCodeOwnIP, 0, room.Players[0]), room.Players[0])
			if err != nil {
				return err
			}
			log.Println("Player", addr, "created room", room)
		}
	default:
		return errors.New("Received unknown message")
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
