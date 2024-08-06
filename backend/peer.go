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
	Choked     bool     `json:"choked"`
	Interested bool     `json:"interested"`
	Bitfield   Bitfield `json:"bitfield"`
	IP         string   `bencode:"ip" json:"ip"`
	Port       string   `bencode:"port" json:"port"`
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

// A Bitfield represents the pieces that a peer has
type Bitfield []byte

func NewBitfield(b []byte) Bitfield {
	return Bitfield(b)
}

// HasPiece tells if a bitfield has a particular index set
func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	return bf[byteIndex]>>(7-offset)&1 != 0
}

// SetPiece sets a bit in the bitfield
func (bf Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8
	bf[byteIndex] |= 1 << (7 - offset)
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
			runtime.EventsEmit(ctx, "peer-disconnect", peer)
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

		handleMessage(conn, msg, peer)
	}
}

func handleMessage(conn net.Conn, msg *Message, peer *Peer) {
	client := GetClient()
	if client == nil {
		panic("client not found")
	}
	switch msg.ID {
	case MsgChoke:
		peer.Choked = true
		fmt.Println("Choked by:", peer.String())
	case MsgUnchoke:
		peer.Choked = false
		fmt.Println("Unchoked by:", peer.String())
		// You might want to start requesting pieces here
	case MsgInterested:
		peer.Interested = true
		fmt.Println("Peer interested:", peer.String())
		// If we're not choking the peer, we might want to unchoke them
	case MsgNotInterested:
		peer.Interested = false
		fmt.Println("Peer not interested:", peer.String())
	case MsgHave:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload)
		peer.Bitfield.SetPiece(int(pieceIndex))
		fmt.Printf("Peer %s has piece %d\n", peer.String(), pieceIndex)
		// You might want to express interest if you need this piece
	case MsgBitfield:
		bf := NewBitfield(msg.Payload)
		peer.Bitfield = bf

		for i := range bf {
			if bf[i] == 1 {
				client.Torrent.havePieces[peer.Identifier()] |= (1 << uint(i))
			}
		}

		fmt.Println("Received bitfield from:", peer.String())
		for i := 0; i < client.Torrent.bencodeTorrent.NumPieces(); i++ {
			if peer.Bitfield.HasPiece(i) && !client.Bitfield.HasPiece(i) {
				// This peer has a piece we need
				fmt.Println("Peer has piece", i, "that we don't have")
				fmt.Println("Sending interested message to:", peer.String())
				sendInterestedMessage(conn)
				break
			}
		}
	case MsgRequest:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		fmt.Printf("Peer %s requested piece %d, begin %d, length %d\n", peer.String(), index, begin, length)
		// Handle the request: check if you have the piece and send it if you do
	case MsgPiece:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		data := msg.Payload[8:]
		fmt.Printf("Received piece %d, begin %d, length %d from %s\n", index, begin, len(data), peer.String())
		// Save this piece data and update your bitfield
	case MsgCancel:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		fmt.Printf("Peer %s cancelled request for piece %d, begin %d, length %d\n", peer.String(), index, begin, length)
		// Remove this piece from your queue of pieces to send, if applicable
	}
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

func (p *Peer) Identifier() int {
	return int(binary.BigEndian.Uint32(net.ParseIP(p.IP).To4()))
}

func sendInterestedMessage(conn net.Conn) {
	msg := Message{ID: MsgInterested}
	_, err := conn.Write(msg.Serialize())
	if err != nil {
		panic(err)
	}
}
