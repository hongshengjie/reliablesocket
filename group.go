package reliablesocket

import (
	"reliablesocket/proto/webpubsub"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type Group struct {
	groupId string
	peers   cmap.ConcurrentMap[string, *Peer]
}

func (g *Group) Send(fromPeerId string, noecho bool, data *webpubsub.MessageData) {
	g.peers.IterCb(func(key string, v *Peer) {
		peerId := key
		if noecho && peerId == fromPeerId {
			return
		}
		v.sendDownStreamDataMessage(data)
	})
}
