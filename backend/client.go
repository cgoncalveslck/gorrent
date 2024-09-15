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

func NewClient(torrent *Torrent) *Client {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	client = &Client{
		Torrent: torrent,
		Bitfield: NewBitfield([]byte(torrent.bencodeTorrent.Info.Pieces)),
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
