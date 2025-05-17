package reliablesocket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reliablesocket/proto/webpubsub"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	peerId         string
	conn           *websocket.Conn
	userId         string
	reconnectToken string
}

func (c *Client) Send(msg *webpubsub.UpstreamMessage) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.Write(context.Background(), websocket.MessageBinary, data)
}
func (c *Client) readLoop() {

	for {
		typ, data, err := c.conn.Read(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(data))
		if typ == websocket.MessageBinary {
			var m webpubsub.DownstreamMessage
			err := proto.Unmarshal(data, &m)
			if err != nil {
				fmt.Println(err)
				return
			}

			if x := m.GetSystemMessage(); x != nil {
				fmt.Println("recive", m.GetSystemMessage())
				c.peerId = x.GetConnectedMessage().GetConnectionId()
				c.userId = x.GetConnectedMessage().GetUserId()
				c.reconnectToken = x.GetConnectedMessage().GetReconnectionToken()

				data, _ := json.Marshal(x.GetConnectedMessage())
				f, _ := os.Create("token.json")
				f.Write(data)
				f.Close()
			}
			if m.GetAckMessage() != nil {
				fmt.Println("recive", m.GetAckMessage())
			}
			if m.GetDataMessage() != nil {
				fmt.Println("recive", m.GetDataMessage())
			}
		}
	}
}

func ReNewClient(peerId string, reconnectToken string) *Client {
	conn, _, err := websocket.Dial(context.Background(), "ws://127.0.0.1:1234/client/hubs/testhub?awps_connection_id="+peerId+"&awps_reconnection_token="+reconnectToken, &websocket.DialOptions{})
	if err != nil {
		panic(err)
	}
	cli := &Client{
		peerId: "",
		conn:   conn,
	}
	go cli.readLoop()
	return cli
}
func NewClient(accessToken string) *Client {
	conn, _, err := websocket.Dial(context.Background(), "ws://127.0.0.1:1234/client/hubs/testhub?access_token="+accessToken, &websocket.DialOptions{})
	if err != nil {
		panic(err)
	}
	cli := &Client{
		peerId: "",
		conn:   conn,
	}
	go cli.readLoop()
	return cli
}
