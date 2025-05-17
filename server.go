package reliablesocket

import (
	"fmt"
	"net/http"
	"reliablesocket/aesutil"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/rs/xid"
)

var hub = &Hub{
	hubId:  "testhub",
	groups: cmap.ConcurrentMap[string, *Group]{},
	peers:  cmap.ConcurrentMap[string, *Peer]{},
}

func Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /client/", startWs)
	mux.HandleFunc("GET /client/hubs/{hubId}", startWs)
	http.ListenAndServe("0.0.0.0:1234", mux)
}

func startWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:         []string{},
		InsecureSkipVerify:   false,
		OriginPatterns:       []string{},
		CompressionMode:      0,
		CompressionThreshold: 0,
	})

	if err != nil {
		panic(err)
	}
	hubId := r.PathValue("hubId")
	if hubId == "" {
		hubId = r.URL.Query().Get("hubId")
	}
	accessToken := r.URL.Query().Get("access_token")
	awps_connection_id := r.URL.Query().Get("awps_connection_id")
	awps_reconnection_token := r.URL.Query().Get("awps_reconnection_token")
	fmt.Println(accessToken, awps_connection_id, awps_reconnection_token)
	if accessToken != "" {
		userId, valid := GetUserId(accessToken)
		if !valid {
			panic("accessToken not valid")
		}
		id := xid.New()
		p := NewPeer(id.String(), userId, conn, hub)
		hub.AddPeer(p)
		p.On("died", func(arg PeerEvent) {
			fmt.Println("remove ", p.PeerId)
			hub.RemovePeer(p.PeerId)
		})
		return
	}
	if awps_connection_id != "" && awps_reconnection_token != "" {
		pidtext, err := aesutil.DecryptFromHex(aesutil.AES_GCM, reconnectionKey, awps_reconnection_token)
		if err != nil {
			panic(err)
		}
		pidsp := strings.Split(string(pidtext), ":")
		if len(pidsp) != 2 {
			panic("token error")
		}
		pid := pidsp[0]
		t := pidsp[1]
		if pid != awps_connection_id {
			panic("pid wrong")
		}
		tt, err := strconv.Atoi(t)
		if err != nil {
			panic(err)
		}
		if tt < int(time.Now().Unix()-3600*24*7) {
			panic("token expired")
		}
		p, ok := hub.peers.Get(awps_connection_id)
		if !ok {
			panic("peer not exist")
		}

		p.conn.Store(conn)
		close(p.recov)
	}
}
