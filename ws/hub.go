package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types sent to clients.
const (
	MsgIdeaAdded   = "idea.added"
	MsgIdeaUpdated = "idea.updated" // count bumped
)

// Message is the envelope sent over the WebSocket.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// client wraps a single WebSocket connection.
type client struct {
	conn    *websocket.Conn
	send    chan []byte
	boardID string
	userID  string
}

// Hub manages all client connections, grouped by boardID.
type Hub struct {
	mu      sync.RWMutex
	boards  map[string]map[*client]struct{} // boardID → set of clients
	reg     chan *client
	unreg   chan *client
	done    chan struct{}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewHub creates and starts a Hub.
func NewHub() *Hub {
	h := &Hub{
		boards: make(map[string]map[*client]struct{}),
		reg:    make(chan *client, 64),
		unreg:  make(chan *client, 64),
		done:   make(chan struct{}),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.reg:
			h.mu.Lock()
			if _, ok := h.boards[c.boardID]; !ok {
				h.boards[c.boardID] = make(map[*client]struct{})
			}
			h.boards[c.boardID][c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unreg:
			h.mu.Lock()
			if clients, ok := h.boards[c.boardID]; ok {
				delete(clients, c)
				if len(clients) == 0 {
					delete(h.boards, c.boardID)
				}
			}
			h.mu.Unlock()
			close(c.send)

		case <-h.done:
			return
		}
	}
}

// Broadcast sends a message to all clients connected to the given board.
func (h *Hub) Broadcast(boardID string, msgType string, data interface{}) {
	payload, err := json.Marshal(Message{Type: msgType, Data: data})
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*client, 0, len(h.boards[boardID]))
	for c := range h.boards[boardID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
			// Slow client — drop the message rather than blocking.
		}
	}
}

// Stop shuts down the hub.
func (h *Hub) Stop() { close(h.done) }

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	boardID := r.URL.Query().Get("board_id")
	userID := r.URL.Query().Get("user_id")
	if boardID == "" || userID == "" {
		http.Error(w, "board_id and user_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	c := &client{
		conn:    conn,
		send:    make(chan []byte, 256),
		boardID: boardID,
		userID:  userID,
	}
	h.reg <- c
	log.Printf("ws client connected: board=%s user=%s", boardID, userID)

	go c.writePump()
	c.readPump(h)
}

// readPump reads from the connection (only to detect close).
func (c *client) readPump(h *Hub) {
	defer func() {
		h.unreg <- c
		c.conn.Close()
		log.Printf("ws client disconnected: board=%s user=%s", c.boardID, c.userID)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// writePump writes outgoing messages and sends periodic pings.
func (c *client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
