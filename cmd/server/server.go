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
	MsgCodeJoin      = byte(8)  // Create or join a netplay room
	MsgCodeOwnIP     = byte(9)  // Get to know your own external IP as well as your player index
	MsgCodePeerIP    = byte(10) // Get the IP of your peer, along with its player index
	MsgCodeHandshake = byte(11) // For both peer to contact each others
)

// Room is a game room where 2 players connect
type Room struct {
	CRC       uint32
	Players   *[]net.Addr
	CreatedAt time.Time
}

// Rooms is the list of rooms
var Rooms []Room

func findRoom(crc uint32, addr net.Addr) *Room {
	for _, r := range Rooms {
		if r.CRC == crc &&
			len(*r.Players) > 0 &&
			r.CreatedAt.After(time.Now().Add(-time.Minute*2)) {
			return &r
		}
	}
	return nil
}

func makeReply(id byte, playerID byte, addr net.Addr) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, id)
	binary.Write(buf, binary.LittleEndian, playerID)
	binary.Write(buf, binary.LittleEndian, []byte(addr.String()))
	binary.Write(buf, binary.LittleEndian, byte(0))
	log.Println(buf.Bytes())
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

	var zeros uint32
	binary.Read(r, binary.LittleEndian, &zeros)

	var code byte
	binary.Read(r, binary.LittleEndian, &code)

	switch code {
	case MsgCodeJoin:
		var crc uint32
		binary.Read(r, binary.LittleEndian, &crc)
		room := findRoom(crc, addr)

		if room != nil {
			*room.Players = append(*room.Players, addr)
			log.Println("Player", addr, "joined room", *room)
			log.Println("We now have", len(*room.Players), "players in room")
			log.Println(*room.Players)
			id := len(*room.Players) - 1
			for _, player := range *room.Players {
				if player == addr {
					// sending own IP to newcomer
					log.Println("-- Replying", "MsgCodeOwnIP", byte(id), addr, "to", player)
					conn.WriteTo(makeReply(MsgCodeOwnIP, byte(id), addr), player)
					// also send the players already in the room to newcomer
					for i, player2 := range *room.Players {
						if player2 != addr {
							log.Println("---- Replying", "MsgCodePeerIP", byte(i), player2, "to", player)
							conn.WriteTo(makeReply(MsgCodePeerIP, byte(i), player2), player)
						}
					}
				} else {
					// send the address of the newcomer to everyone in the room
					log.Println("-- Replying", "MsgCodePeerIP", byte(id), addr, "to", player)
					conn.WriteTo(makeReply(MsgCodePeerIP, byte(id), addr), player)
				}
			}
		} else {
			room := Room{
				CRC:       crc,
				Players:   &[]net.Addr{addr},
				CreatedAt: time.Now(),
			}
			Rooms = append(Rooms, room)
			log.Println("Player", addr, "created room", room)
			log.Println("We now have", len(*room.Players), "players in room")
			log.Println("-- Replying", "MsgCodeOwnIP", 0, addr, "to", addr)
			_, err := conn.WriteTo(makeReply(MsgCodeOwnIP, 0, addr), addr)
			if err != nil {
				return err
			}
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
