package backend

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

	"github.com/jackpal/bencode-go"
)

type BencodeTorrent struct {
	Announce string      `bencode:"announce" json:"announce"`
	Info     bencodeInfo `bencode:"info" json:"info"`
}

func (bT *BencodeTorrent) VerifyPiece(index uint32, data []byte) bool {
	// Verify that the piece at the given index matches the hash in the torrent file
	h := sha1.New()
	h.Write(data)
	hash := h.Sum(nil)

	start := index * 20
	end := start + 20
	return bytes.Equal(hash, []byte(bT.Info.Pieces[start:end]))
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces" json:"-"`
	PieceLength int    `bencode:"piece length" json:"-"`
	Length      int    `bencode:"length" json:"-"`
	Name        string `bencode:"name" json:"name"`
}

type Torrent struct {
	bencodeTorrent *BencodeTorrent
	havePieces     Bitfield
}

func (bI *bencodeInfo) hash() [20]byte {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *bI)
	if err != nil {
		panic(err)
	}
	h := sha1.Sum(buf.Bytes())
	return h
}

func (bT *BencodeTorrent) NumPieces() int {
	pieceHash := []byte(bT.Info.Pieces)
	return len(pieceHash) / 20 // Each piece hash is 20 bytes
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

func HandleFile(ctx context.Context, path string) (*BencodeTorrent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	r := bufio.NewReader(f)
	bcode, err := getBencode(r)
	if err != nil {
		return nil, err
	}

	readTorrentFile(ctx, bcode)
	return bcode, nil
}

func readTorrentFile(ctx context.Context, bcode *BencodeTorrent) {
	str := "-TX0001-7478636c636b"

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
		go ConnectToPeer(ctx, peer, bcode.Info.hash(), peerID)
	}

}

func getBencode(r io.Reader) (*BencodeTorrent, error) {
	bto := BencodeTorrent{}
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

func getTrackerURL(b *BencodeTorrent, peerID string) (string, error) {
	base, err := url.Parse(b.Announce)
	if err != nil {
		return "", err
	}

	infoHash := b.Info.hash()
	params := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{peerID[:]},
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
			IP:               ip,
			Port:             strconv.Itoa(int(port)),
			PeerChoked:       true,
			ClientInterested: false,
			ClientChoked:     true,
			PeerInterested:   false,
		})
	}

	return peers, nil
}
