// backend/client.go
package backend

import (
	"sync"
)

var (
	client      *Client
	clientMutex sync.Mutex
)

type Client struct {
	Torrent  *Torrent
	Peers    []*Peer
	Bitfield Bitfield
	mutex    sync.Mutex
}

func NewClient(torrent *BencodeTorrent) *Client {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	client = &Client{
		Torrent: &Torrent{
			bencodeTorrent: torrent,
			havePieces:     NewBitfield(nil),
		},
		Bitfield: NewBitfield([]byte(torrent.Info.Pieces)),
	}
	return client
}

func (c *Client) AddPeer(peer *Peer) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Peers = append(c.Peers, peer)
}

func GetClient() *Client {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	return client
}
