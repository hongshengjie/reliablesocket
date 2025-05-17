package reliablesocket

import (
	"context"
	"fmt"
	"reliablesocket/aesutil"
	"reliablesocket/events"
	"reliablesocket/proto/webpubsub"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const reconnectionKey = "reconnectionKey"

const peerStatusAlive = 0
const peerStatusWaitReconnect = 1
const peerStatusDied = 2

type PeerEvent struct {
	EventMessage       *webpubsub.UpstreamMessage_EventMessage
	JoinGroupMessage   *webpubsub.UpstreamMessage_JoinGroupMessage
	LeaveGroupMessage  *webpubsub.UpstreamMessage_LeaveGroupMessage
	SendToGroupMessage *webpubsub.UpstreamMessage_SendToGroupMessage
	SequenceAckMessage *webpubsub.UpstreamMessage_SequenceAckMessage
}
type Peer struct {
	PeerId string
	status *atomic.Int32
	conn   *atomic.Value
	events.EventEmmiter[PeerEvent]
	group *Group
	hub   *Hub
	recov chan struct{}
}

func NewPeer(id, userId string, conn *websocket.Conn, hub *Hub) *Peer {
	p := &Peer{
		PeerId:       id,
		conn:         &atomic.Value{},
		status:       &atomic.Int32{},
		EventEmmiter: events.New[PeerEvent](),
		hub:          hub,
		recov:        make(chan struct{}),
	}
	p.conn.Store(conn)
	go p.readLoop()
	plaintext := fmt.Sprintf("%s:%d", p.PeerId, time.Now().Unix())
	reconnectionToken, _ := aesutil.EncryptToHex(aesutil.AES_GCM, reconnectionKey, []byte(plaintext))

	p.sendDownStreamSystemMessage(&webpubsub.DownstreamMessage_SystemMessage{
		Message: &webpubsub.DownstreamMessage_SystemMessage_ConnectedMessage_{ConnectedMessage: &webpubsub.DownstreamMessage_SystemMessage_ConnectedMessage{
			ConnectionId:      id,
			UserId:            userId,
			ReconnectionToken: reconnectionToken,
		}},
	})
	return p
}
func (p *Peer) Close() {
	if p.status.Load() != peerStatusAlive {
		return
	}
	if p.status.Load() == peerStatusAlive {
		p.status.CompareAndSwap(peerStatusAlive, peerStatusWaitReconnect)
		p.Emit("waitreconnect", PeerEvent{})
		go func() {
			select {
			case <-time.After(time.Second * 30):
				p.status.CompareAndSwap(peerStatusWaitReconnect, peerStatusDied)
				p.Emit("died", PeerEvent{})
				fmt.Println("died")
			case <-p.recov:
				p.recov = make(chan struct{})
				p.status.CompareAndSwap(peerStatusWaitReconnect, peerStatusAlive)
				go p.readLoop()
				p.Emit("alive", PeerEvent{})
				fmt.Println("reconnected")
			}
		}()
	}
}
func (p *Peer) readLoop() {
	var e error
	defer func() {
		if e != nil {
			fmt.Println(e)
		}
		p.Close()
	}()
	for {
		if p.status.Load() != peerStatusAlive {
			return
		}
		connA := p.conn.Load()
		conn := connA.(*websocket.Conn)
		msgType, data, err := conn.Read(context.Background())
		if err != nil {
			e = err
			return
		}
		if msgType == websocket.MessageBinary {
			var m webpubsub.UpstreamMessage
			if err := proto.Unmarshal(data, &m); err != nil {
				e = err
				return
			}
			fmt.Println(&m)
			if x := m.GetEventMessage(); x != nil {
				p.Emit("event", PeerEvent{EventMessage: x})
				fmt.Println(x)
			}
			if x := m.GetJoinGroupMessage(); x != nil {
				p.Emit("joingroup", PeerEvent{JoinGroupMessage: x})
				fmt.Println(x)
				group := x.GetGroup()
				p.hub.JoinGroup(group, p.PeerId)

				if x.GetAckId() != 0 {
					p.sendDownStreamAckMessage(&webpubsub.DownstreamMessage_AckMessage{
						AckId:   x.GetAckId(),
						Success: true,
					})
				}
			}
			if x := m.GetLeaveGroupMessage(); x != nil {
				p.Emit("leavegroup", PeerEvent{LeaveGroupMessage: x})
				fmt.Println(x)
				p.hub.LeaveGroup(x.GetGroup(), p.PeerId)

				if x.GetAckId() != 0 {
					p.sendDownStreamAckMessage(&webpubsub.DownstreamMessage_AckMessage{
						AckId:   x.GetAckId(),
						Success: true,
					})
				}
			}
			if x := m.GetSendToGroupMessage(); x != nil {
				p.Emit("sendtogroup", PeerEvent{SendToGroupMessage: x})
				fmt.Println(x)
				if p.group != nil && p.group.groupId == x.Group {
					var noecho bool
					if x.NoEcho != nil {
						noecho = *x.NoEcho
					}
					p.group.Send(p.PeerId, noecho, x.Data)
				}
				if x.GetAckId() != 0 {
					p.sendDownStreamAckMessage(&webpubsub.DownstreamMessage_AckMessage{
						AckId:   x.GetAckId(),
						Success: true,
					})
				}
			}
			if x := m.GetSequenceAckMessage(); x != nil {
				p.Emit("sequenceack", PeerEvent{SequenceAckMessage: x})
				fmt.Println(x)
			}
		}
	}
}
func (p *Peer) sendTextMessage(text string) error {
	msg2 := &webpubsub.MessageData{Data: &webpubsub.MessageData_TextData{TextData: text}}
	return p.sendDownStreamDataMessage(msg2)
}

func (p *Peer) sendJSONMessage(msg string) error {
	msg2 := &webpubsub.MessageData{Data: &webpubsub.MessageData_JsonData{JsonData: msg}}
	return p.sendDownStreamDataMessage(msg2)
}

func (p *Peer) sendBinaryMessge(msg []byte) error {
	msg2 := &webpubsub.MessageData{Data: &webpubsub.MessageData_BinaryData{BinaryData: msg}}
	return p.sendDownStreamDataMessage(msg2)
}

func (p *Peer) sendProtobufMessage(msg proto.Message) error {
	mm, err := anypb.New(msg)
	if err != nil {
		return err
	}
	msg2 := &webpubsub.MessageData{Data: &webpubsub.MessageData_ProtobufData{ProtobufData: mm}}
	return p.sendDownStreamDataMessage(msg2)
}

func (p *Peer) sendDownStreamSystemMessage(msg *webpubsub.DownstreamMessage_SystemMessage) error {
	msg2 := &webpubsub.DownstreamMessage{
		Message: &webpubsub.DownstreamMessage_SystemMessage_{SystemMessage: msg}}
	data, err := proto.Marshal(msg2)
	if err != nil {
		return err
	}
	return p.sendToPeer(data)

}
func (p *Peer) sendDownStreamAckMessage(msg *webpubsub.DownstreamMessage_AckMessage) error {
	msg2 := &webpubsub.DownstreamMessage{
		Message: &webpubsub.DownstreamMessage_AckMessage_{AckMessage: msg}}
	data, err := proto.Marshal(msg2)
	if err != nil {
		return err
	}
	return p.sendToPeer(data)

}
func (p *Peer) sendDownStreamDataMessage(msg *webpubsub.MessageData) error {
	msg2 := &webpubsub.DownstreamMessage{
		Message: &webpubsub.DownstreamMessage_DataMessage_{DataMessage: &webpubsub.DownstreamMessage_DataMessage{
			Data: msg}}}
	data, err := proto.Marshal(msg2)
	if err != nil {
		return err
	}
	return p.sendToPeer(data)

}
func (p *Peer) sendToPeer(data []byte) error {
	pp := p.conn.Load()
	ppp := pp.(*websocket.Conn)
	return ppp.Write(context.Background(), websocket.MessageBinary, data)
}
