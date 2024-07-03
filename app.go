package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

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

// Message stores ID and payload of a message
type Message struct {
	ID      messageID
	Payload []byte
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

type Handshake struct {
	// 20-byte SHA1 hash of the info key in the metainfo file. This is the same info_hash that is transmitted in tracker requests.
	info_hash [20]byte
	// 20-byte string used as a unique ID for the client. This is usually the same peer_id that is transmitted in tracker requests.
	peer_id [20]byte
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, 68)
	buf[0] = 19
	copy(buf[1:20], "BitTorrent protocol")
	copy(buf[28:48], h.info_hash[:])
	copy(buf[48:68], h.peer_id[:])
	return buf
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
	copy(hs.info_hash[:], buf[28:48])
	copy(hs.peer_id[:], buf[48:68])

	return &hs, nil
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}
type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

func (i *bencodeInfo) hash() [20]byte {
	var buf bytes.Buffer
	bencode.Marshal(&buf, *i)
	h := sha1.Sum(buf.Bytes())
	return h
}

type TrackerResponse struct {
	FailureReason  string `bencode:"failure reason,omitempty"`
	WarningMessage string `bencode:"warning message,omitempty"`
	Interval       int    `bencode:"interval"`
	MinInterval    int    `bencode:"min interval,omitempty"`
	TrackerID      string `bencode:"tracker id,omitempty"`
	Complete       int    `bencode:"complete"`
	Incomplete     int    `bencode:"incomplete"`
	Peers          string `bencode:"peers"`
}

type Peer struct {
	IP   string `bencode:"ip"`
	Port string `bencode:"port"`
}

func (a *App) OpenFileDialog() {
	options := runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Torrent Files",
				Pattern:     "*.torrent",
			},
		},
	}

	path, err := runtime.OpenFileDialog(a.ctx, options)
	if err != nil {
		panic(err)
	}

	readTorrentFile(path)
}

func readTorrentFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(f)
	bcode, err := getBencode(r)
	if err != nil {
		panic(err)
	}

	str := "-TX0001-7478636c636b"
	if len(str) != 20 {
		fmt.Println("Error: String length is not 20")
		return
	}

	var peerID [20]byte
	copy(peerID[:], str)

	trackerUrl, err := getTrackerURL(bcode, str)
	if err != nil {
		panic(err)
	}

	resp, err := http.Get(trackerUrl)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	tr, err := getTracker(resp.Body)
	if err != nil {
		panic(err)
	}

	peers, err := parseBinaryPeers(tr.Peers)
	if err != nil {
		panic(err)
	}

	for _, peer := range peers {
		go connectToPeer(peer, bcode.Info.hash(), peerID)
	}

}

func connectToPeer(peer *Peer, infoHash, peerID [20]byte) {
	var conn net.Conn
	var err error

	conn, err = net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	hs := &Handshake{
		info_hash: infoHash,
		peer_id:   peerID,
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

	if !bytes.Equal(receivedHS.info_hash[:], infoHash[:]) {
		fmt.Println("info hash mismatch")
		return
	}

	fmt.Printf("Successfully connected to peer: %s\n", peer.String())

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
	}
}

func (p *Peer) String() string {
	return strings.Join([]string{p.IP, p.Port}, ":")
}

func getBencode(r io.Reader) (*bencodeTorrent, error) {
	bto := bencodeTorrent{}
	err := bencode.Unmarshal(r, &bto)
	if err != nil {
		return nil, err
	}
	return &bto, nil
}

func getTracker(r io.Reader) (*TrackerResponse, error) {
	bto := TrackerResponse{}
	err := bencode.Unmarshal(r, &bto)
	if err != nil {
		return nil, err
	}
	return &bto, nil
}

func getTrackerURL(b *bencodeTorrent, peerID string) (string, error) {
	base, err := url.Parse(b.Announce)
	if err != nil {
		return "", err
	}

	infoHash := b.Info.hash()
	params := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{base.Port()},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(b.Info.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}

func parseBinaryPeers(data string) ([]*Peer, error) {
	const peerSize = 6 // 4 bytes for IP and 2 bytes for port
	bytesData := []byte(data)

	if len(bytesData)%peerSize != 0 {
		return nil, fmt.Errorf("invalid binary peers length")
	}

	var peers []*Peer
	for i := 0; i < len(bytesData); i += peerSize {
		ip := net.IP(bytesData[i : i+4]).String()
		port := binary.BigEndian.Uint16(bytesData[i+4 : i+6])
		peers = append(peers, &Peer{
			IP:   ip,
			Port: strconv.Itoa(int(port)),
		})
	}

	return peers, nil
}
