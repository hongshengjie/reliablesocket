package reliablesocket

import (
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Hub struct {
	hubId  string
	groups cmap.ConcurrentMap[string, *Group]
	peers  cmap.ConcurrentMap[string, *Peer]
}

func (h *Hub) AddPeer(p *Peer) {
	h.peers.Set(p.PeerId, p)
}

func (h *Hub) RemovePeer(peerId string) {
	h.peers.Remove(peerId)
}

func (h *Hub) JoinGroup(groupId, peerId string) {
	p, ok := h.peers.Get(peerId)

	if ok {
		pp := p
		if pp.group != nil && pp.group.groupId == groupId {
			return
		}
		if pp.group != nil {
			pp.group.peers.Remove(peerId)
		}
		g, ok2 := h.groups.Get(groupId)
		if ok2 {
			gg := g
			gg.peers.Set(peerId, p)
			pp.group = gg
		} else {
			gg := &Group{groupId: groupId, peers: cmap.ConcurrentMap[string, *Peer]{}}
			gg.peers.Set(peerId, p)
			h.groups.Set(groupId, gg)
			pp.group = gg
		}
	}
}

func (h *Hub) LeaveGroup(groupId, peerId string) {
	g, ok := h.groups.Get(groupId)
	if ok {
		g.peers.Remove(peerId)
	}
}
