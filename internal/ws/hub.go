package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type RedisClient interface {
	Publish(ctx context.Context, channel string, msg []byte) error
	Subscribe(ctx context.Context, channel string) Subscription
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
}

type Subscription interface {
	Channel() <-chan Message
	Close() error
}

type Message struct {
	Channel string
	Payload string
}

type Hub struct {
	mu          sync.RWMutex
	boards      map[string]map[string]*Client
	sequences   map[string]*int64
	redisClient RedisClient
	events      chan *Event
	done        chan struct{}
	subs        map[string]Subscription
}

func NewHub(redisClient RedisClient) *Hub {
	return &Hub{
		boards:      make(map[string]map[string]*Client),
		sequences:   make(map[string]*int64),
		redisClient: redisClient,
		events:      make(chan *Event, 4096),
		done:        make(chan struct{}),
		subs:        make(map[string]Subscription),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	bid := client.boardID
	if _, ok := h.boards[bid]; !ok {
		h.boards[bid] = make(map[string]*Client)
		seq := new(int64)
		h.sequences[bid] = seq

		channel := fmt.Sprintf("board:%s", bid)
		sub := h.redisClient.Subscribe(context.Background(), channel)
		h.subs[bid] = sub
		go h.listenRedisChannel(bid, sub)
	}
	h.boards[bid][client.userID] = client

	log.Printf("ws client registered: user=%s board=%s", client.userID, client.boardID)

	if client.lastSeq > 0 {
		go h.replayMissed(client)
	}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	bid := client.boardID
	if clients, ok := h.boards[bid]; ok {
		delete(clients, client.userID)
		if len(clients) == 0 {
			delete(h.boards, bid)
			delete(h.sequences, bid)
			if sub, ok := h.subs[bid]; ok {
				sub.Close()
				delete(h.subs, bid)
			}
		}
	}
	log.Printf("ws client unregistered: user=%s board=%s", client.userID, client.boardID)
}

func (h *Hub) Broadcast(boardID string, event *Event) {
	h.mu.RLock()
	clients := h.boards[boardID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	seq := atomic.AddInt64(h.sequences[boardID], 1)
	event.SequenceID = seq
	event.Timestamp = time.Now().UTC()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("marshal event error: %v", err)
		return
	}

	for _, client := range clients {
		client.Send(data)
		client.SetLastSeq(seq)
	}
}

func (h *Hub) PublishAndBroadcast(ctx context.Context, boardID string, event *Event) {
	seq := atomic.AddInt64(h.sequences[boardID], 1)
	event.SequenceID = seq
	event.Timestamp = time.Now().UTC()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("marshal event error: %v", err)
		return
	}

	channel := fmt.Sprintf("board:%s", boardID)
	if err := h.redisClient.Publish(ctx, channel, data); err != nil {
		log.Printf("redis publish error: %v", err)
	}

	h.mu.RLock()
	clients := h.boards[boardID]
	for _, client := range clients {
		client.Send(data)
		client.SetLastSeq(seq)
	}
	h.mu.RUnlock()
}

func (h *Hub) HandleClientMessage(client *Client, eventType EventType, payload json.RawMessage) {
	switch eventType {
	case EventCursorMove:
		h.Broadcast(client.boardID, &Event{
			BoardID: client.boardID,
			Type:    EventCursorMove,
			ActorID: client.userID,
			Data:    payload,
		})
	default:
	}
}

func (h *Hub) listenRedisChannel(boardID string, sub Subscription) {
	ch := sub.Channel()
	for msg := range ch {
		var event Event
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("redis unmarshal event error: %v", err)
			continue
		}

		h.mu.RLock()
		clients := h.boards[boardID]
		for _, client := range clients {
			if event.SequenceID > client.LastSeq() {
				client.Send([]byte(msg.Payload))
				client.SetLastSeq(event.SequenceID)
			}
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) replayMissed(client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	channel := fmt.Sprintf("board:%s:history", client.boardID)
	missed, err := h.redisClient.LRange(ctx, channel, 0, -1)
	if err != nil {
		log.Printf("replay: failed to get history for board=%s: %v", client.boardID, err)
		return
	}

	for _, msg := range missed {
		var event Event
		if err := json.Unmarshal([]byte(msg), &event); err != nil {
			continue
		}
		if event.SequenceID > client.lastSeq {
			client.Send([]byte(msg))
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sub := range h.subs {
		sub.Close()
	}
}
