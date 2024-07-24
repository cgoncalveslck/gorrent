## Links

[Bittorrent Protocol Specification v1.0](https://wiki.theory.org/BitTorrentSpecification)

[How to make your own bittorrent client - Allen Kim](https://allenkim67.github.io/programming/2016/05/04/how-to-make-your-own-bittorrent-client.html)

[How to Write a Bittorrent Client Part 2 - Kristen Widman](http://www.kristenwidman.com/blog/71/how-to-write-a-bittorrent-client-part-2/)

[The BitTorrent Protocol - Joe?
](https://www.morehawes.ca/uk/old-guides/the-bittorrent-protocol)

[Building a BitTorrent client from the ground up in Go - Jesse Li](https://blog.jse.li/posts/torrent/)
____
## Flow

From my understanding, (broadly) the flow of requests should be something like:

↔ Handshake \
← Bitfield \
→ Interested \
← Unchoke \
→ Request \
← Piece 

## Peers
#### Client connections start out as "choked" and "not interested". In other words:

am_choking = 1\
am_interested = 0\
peer_choking = 1\
peer_interested = 0


```go
peers = append(peers, &Peer{
  IP:          ip,
  Port:        strconv.Itoa(int(port)),
  Choked:      true,
  Interesting: false,
  Interested:  false,
  Choking:     true,
})
// backend/torrent.go:161
```
