package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reliablesocket"
	"reliablesocket/proto/webpubsub"
	"sync/atomic"
)

var ackId = &atomic.Int64{}
var join bool

func main() {
	data, _ := os.ReadFile("token.json")
	var d webpubsub.DownstreamMessage_SystemMessage_ConnectedMessage
	json.Unmarshal(data, &d)
	var cli *reliablesocket.Client
	if d.GetReconnectionToken() != "" {
		fmt.Println(d.String())
		cli = reliablesocket.ReNewClient(d.GetConnectionId(), d.GetReconnectionToken())
	} else {
		cli = reliablesocket.NewClient("bob")
	}
	if !join {
		id2 := ackId.Add(1)
		err := cli.Send(&webpubsub.UpstreamMessage{Message: &webpubsub.UpstreamMessage_JoinGroupMessage_{JoinGroupMessage: &webpubsub.UpstreamMessage_JoinGroupMessage{
			Group: "golang",
			AckId: &id2,
		}}})
		join = true
		if err != nil {
			panic(err)
		}
	}

	if join {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()

			id3 := ackId.Add(1)
			noecho := true
			cli.Send(&webpubsub.UpstreamMessage{
				Message: &webpubsub.UpstreamMessage_SendToGroupMessage_{SendToGroupMessage: &webpubsub.UpstreamMessage_SendToGroupMessage{Group: "golang", AckId: &id3, NoEcho: &noecho, Data: &webpubsub.MessageData{Data: &webpubsub.MessageData_TextData{TextData: line}}}},
			})
		}
	}
}
