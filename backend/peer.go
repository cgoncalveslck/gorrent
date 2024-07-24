package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type messageID uint8

const (
	MsgChoke         messageID = 0
	MsgUnchoke       messageID = 1
	MsgInterested    messageID = 2
	MsgNotInterested messageID = 3
	MsgHave          messageID = 4
	MsgBitfield      messageID = 5
	MsgRequest       messageID = 6
	MsgPiece         messageID = 7
	MsgCancel        messageID = 8
)

type Connection struct {
	Conn     net.Conn
	PeerID   [20]byte
	InfoHash [20]byte
	Peer     *Peer
}

type Message struct {
	ID      messageID
	Payload []byte
}

type Peer struct {
	IP   string `bencode:"ip" json:"ip"`
	Port string `bencode:"port" json:"port"`
}

type Handshake struct {
	// 20-byte SHA1 hash of the info key in the metainfo file. This is the same info_hash that is transmitted in tracker requests.
	infoHash [20]byte
	// 20-byte string used as a unique ID for the client. This is usually the same peer_id that is transmitted in tracker requests.
	peerID [20]byte
}

func Read(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	m := Message{
		ID:      messageID(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return &m, nil
}

func parseHandshake(buf []byte) (*Handshake, error) {
	if len(buf) != 68 {
		return nil, fmt.Errorf("invalid handshake length: expected 68 bytes, got %d", len(buf))
	}

	if buf[0] != 19 {
		return nil, fmt.Errorf("invalid protocol length: expected 19, got %d", buf[0])
	}

	protocol := string(buf[1:20])
	if protocol != "BitTorrent protocol" {
		return nil, fmt.Errorf("invalid protocol: expected 'BitTorrent protocol', got '%s'", protocol)
	}

	var hs Handshake
	copy(hs.infoHash[:], buf[28:48])
	copy(hs.peerID[:], buf[48:68])

	return &hs, nil
}

func ConnectToPeer(ctx context.Context, peer *Peer, infoHash, peerID [20]byte) {
	var conn net.Conn
	var err error

	conn, err = net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return
	}

	hs := &Handshake{
		infoHash: infoHash,
		peerID:   peerID,
	}

	serial := hs.Serialize()
	_, err = conn.Write(serial)
	if err != nil {
		fmt.Println(err)
		return
	}

	buf := make([]byte, 68) // buffer for handshake response
	n2, err := conn.Read(buf)
	if err != nil {
		return
	}

	// Parse the handshake from the buffer
	receivedHS, err := parseHandshake(buf[:n2])
	if err != nil {
		return
	}

	if !bytes.Equal(receivedHS.infoHash[:], infoHash[:]) {
		fmt.Println("info hash mismatch")
		return
	}

	runtime.EventsEmit(ctx, "peer-connect", peer)
	fmt.Printf("Successfully connected to peer: %s\n", peer.String())

	interestedMsg := Message{ID: MsgInterested}
	_, err = conn.Write(interestedMsg.Serialize())
	if err != nil {
		panic(err)
	}

	unchokeMsg := Message{ID: MsgUnchoke}
	_, err = conn.Write(unchokeMsg.Serialize())
	if err != nil {
		panic(err)
	}

	for {
		msg, err := Read(conn)

		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by peer:", peer.String())
			} else {
				fmt.Println("Error reading message:", err)
			}
			return
		}

		if msg == nil {
			fmt.Println("Received keep-alive message from:", peer.String())
			continue
		}

		// Process the message here
		fmt.Println("Received message: ID=", msg.ID, " from: ", peer.String())

		if msg.ID == MsgBitfield {

		}
	}
}

func (c *Connection) Read() (*Message, error) {
	msg, err := Read(c.Conn)
	return msg, err
}

func (c *Connection) SendRequest(index, begin, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	req := &Message{ID: MsgRequest, Payload: payload}
	_, err := c.Conn.Write(req.Serialize())
	return err
}

// SendMessage sends a message to the peer
func (c *Connection) SendMessage(id messageID) error {
	msg := Message{ID: id}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendHave sends a Have message to the peer
func (c *Connection) SendHave(index int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))

	msg := &Message{ID: MsgHave, Payload: payload}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1) // +1 for id
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, 68)
	buf[0] = 19
	copy(buf[1:20], "BitTorrent protocol")
	copy(buf[28:48], h.infoHash[:])
	copy(buf[48:68], h.peerID[:])
	return buf
}

func (p *Peer) String() string {
	return strings.Join([]string{p.IP, p.Port}, ":")
}
