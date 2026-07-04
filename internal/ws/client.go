package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	userID   string
	boardID  string
	lastSeq  int64
	send     chan []byte
	done     chan struct{}
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, boardID string, lastSeq int64) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		userID:  userID,
		boardID: boardID,
		lastSeq: lastSeq,
		send:    make(chan []byte, 256),
		done:    make(chan struct{}),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read error (user=%s): %v", c.userID, err)
			}
			break
		}

		var envelope struct {
			Type    EventType       `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			log.Printf("ws invalid message from user=%s: %v", c.userID, err)
			continue
		}

		c.hub.HandleClientMessage(c, envelope.Type, envelope.Payload)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("ws write error (user=%s): %v", c.userID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		log.Printf("ws client send buffer full, dropping message for user=%s", c.userID)
	}
}

func (c *Client) Close() {
	close(c.done)
}

func (c *Client) UserID() string { return c.userID }
func (c *Client) BoardID() string { return c.boardID }
func (c *Client) LastSeq() int64 { return c.lastSeq }
func (c *Client) SetLastSeq(seq int64) { c.lastSeq = seq }
