package ws

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type HubProvider interface {
	Register(client *Client)
}

func ServeWS(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boardID := r.URL.Query().Get("board_id")
		userID := r.URL.Query().Get("user_id")
		lastSeqStr := r.URL.Query().Get("last_seq")

		if boardID == "" || userID == "" {
			http.Error(w, "board_id and user_id required", http.StatusBadRequest)
			return
		}

		var lastSeq int64
		if lastSeqStr != "" {
			lastSeq, _ = strconv.ParseInt(lastSeqStr, 10, 64)
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		client := NewClient(hub, conn, userID, boardID, lastSeq)
		hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	}
}

func BoardIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "boards" {
		return parts[1]
	}
	return ""
}
