package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
	// Serialize handles length prefix
	ID      messageID
	Payload []byte
}

type Peer struct {
	ClientChoked     bool     `json:"client_choked"`
	PeerChoked       bool     `json:"peer_choked"`
	ClientInterested bool     `json:"client_interested"`
	PeerInterested   bool     `json:"peer_interested"`
	Bitfield         Bitfield `json:"bitfield"`
	IP               string   `bencode:"ip" json:"ip"`
	Port             string   `bencode:"port" json:"port"`
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

func printPeerStatus(peer *Peer) {
	clientChokedStatus := "not choked"
	if peer.ClientChoked {
		clientChokedStatus = "choked"
	}

	peerChokedStatus := "not choked"
	if peer.PeerChoked {
		peerChokedStatus = "choked"
	}

	clientInterestedStatus := "not interested"
	if peer.ClientInterested {
		clientInterestedStatus = "interested"
	}

	peerInterestedStatus := "not interested"
	if peer.PeerInterested {
		peerInterestedStatus = "interested"
	}

	fmt.Printf("Peer is: %s and %s \n Client is: %s and %s\n",
		peerInterestedStatus, peerChokedStatus, clientInterestedStatus, clientChokedStatus)
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

	// fmt.Println("Connecting to peer:", peer.String())
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

	// fmt.Println("PEER STRING", string(receivedHS.peerID[:]))
	if !bytes.Equal(receivedHS.infoHash[:], infoHash[:]) {
		fmt.Println("info hash mismatch")
		return
	}

	runtime.EventsEmit(ctx, "peer-connect", peer)
	cl := GetClient()
	if cl == nil {
		panic("client not found")
	}
	cl.AddPeer(peer)

	err = peer.SendMessage(conn, MsgUnchoke, nil)
	if err != nil {
		fmt.Println("Error sending unchoke message", err)
		return
	}
	peer.PeerChoked = false

	err = peer.SendMessage(conn, MsgInterested, nil)
	if err != nil {
		fmt.Println("Error sending interested message", err)
		return
	}
	peer.ClientInterested = true

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
	cl := GetClient()
	if cl == nil {
		panic("client not found")
	}

	switch msg.ID {
	case MsgChoke:
		peer.ClientChoked = true
		// fmt.Println("Choked by:", peer.String())
	case MsgUnchoke:
		peer.ClientChoked = false

		// Constants
		const BlockSize = 16384 // 16 KB

		// Send multiple requests for the first piece
		pieceIndex := 0
		pieceLength := client.Torrent.bencodeTorrent.Info.PieceLength
		numBlocks := (pieceLength + BlockSize - 1) / BlockSize // Round up division

		for blockIndex := 0; blockIndex < numBlocks; blockIndex++ {
			begin := blockIndex * BlockSize
			length := BlockSize
			if begin+length > pieceLength {
				length = pieceLength - begin // Last block might be smaller
			}

			fmt.Printf("Requesting piece %d, block %d, begin %d, length %d from %s\n",
				pieceIndex, blockIndex, begin, length, peer.String())

			err := peer.SendRequest(conn, pieceIndex, begin, length)
			if err != nil {
				fmt.Println("Error sending request", err)
				// Don't panic, just log and continue
				log.Println(err)
				continue
			}

			fmt.Printf("Sent request for piece %d, block %d to: %s\n", pieceIndex, blockIndex, peer.String())
		}

	case MsgInterested:
		peer.PeerInterested = true
		// fmt.Println("Peer interested:", peer.String())
		// If we're not choking the peer, we might want to unchoke them
	case MsgNotInterested:
		peer.PeerInterested = false
		// fmt.Println("Peer not interested:", peer.String())
	case MsgHave:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload)
		peer.Bitfield.SetPiece(int(pieceIndex))
		// fmt.Printf("Peer %s has piece %d\n", peer.String(), pieceIndex)
		// You might want to express interest if you need this piece
	case MsgBitfield:
		bf := NewBitfield(msg.Payload)
		peer.Bitfield = bf

		for i := range bf {
			if bf[i] == 1 {
				client.Torrent.havePieces[peer.Identifier()] |= 1 << uint(i)
			}
		}

		for i := 0; i < client.Torrent.bencodeTorrent.NumPieces(); i++ {
			if peer.Bitfield.HasPiece(i) && !client.Bitfield.HasPiece(i) {
				// This peer has a piece we need

				// send interested message
				if !peer.ClientInterested {
					err := peer.SendMessage(conn, MsgInterested, nil)
					if err != nil {
						fmt.Println("Error sending interested message", err)
						return
					}
					peer.ClientInterested = true
					return
				}
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

		// check if the piece is valid
		if !client.Torrent.bencodeTorrent.VerifyPiece(index, data) {
			err := conn.Close()
			if err != nil {
				panic(err)
			}
			fmt.Println("Invalid piece")
			return
		}

		fmt.Printf("Received piece %d, begin %d, length %d from %s\n", index, begin, len(data), peer.String())

	case MsgCancel:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		fmt.Printf("Peer %s cancelled request for piece %d, begin %d, length %d\n", peer.String(), index, begin, length)
		// Remove this piece from your queue of pieces to send, if applicable
	}
}

func (p *Peer) SendRequest(c net.Conn, index, begin, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	// fmt.Printf("Requesting piece %d, begin %d, length %d from %s\n", index, begin, length, p.String())

	err := p.SendMessage(c, MsgRequest, payload)
	return err
}

// SendMessage sends a message to the peer
func (p *Peer) SendMessage(c net.Conn, id messageID, payload []byte) error {
	var msg Message
	if payload == nil {
		msg = Message{ID: id}
	} else {
		msg = Message{ID: id, Payload: payload}
	}
	_, err := c.Write(msg.Serialize())
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
