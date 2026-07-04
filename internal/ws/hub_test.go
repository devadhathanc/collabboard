package ws

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type fakeRedis struct {
	mu        sync.Mutex
	published []string
	subs      map[string][]chan Message
	history   map[string][]string
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{
		subs:    make(map[string][]chan Message),
		history: make(map[string][]string),
	}
}

func (f *fakeRedis) Publish(_ context.Context, channel string, msg []byte) error {
	f.mu.Lock()
	f.published = append(f.published, string(msg))
	f.mu.Unlock()
	return nil
}

func (f *fakeRedis) Subscribe(_ context.Context, channel string) Subscription {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan Message, 256)
	f.subs[channel] = append(f.subs[channel], ch)
	return &fakeSub{ch: ch}
}

func (f *fakeRedis) LRange(_ context.Context, key string, _, _ int64) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.history[key], nil
}

type fakeSub struct {
	ch chan Message
}

func (s *fakeSub) Channel() <-chan Message  { return s.ch }
func (s *fakeSub) Close() error              { return nil }

func TestHubRegisterAndBroadcast(t *testing.T) {
	fr := newFakeRedis()
	hub := NewHub(fr)

	sendCh := make(chan []byte, 256)
	client := &Client{
		hub:     hub,
		userID:  "user1",
		boardID: "board1",
		lastSeq: 0,
		send:    sendCh,
		done:    make(chan struct{}),
	}

	hub.Register(client)

	evt := &Event{
		BoardID: "board1",
		Type:    EventTaskCreated,
		ActorID: "user1",
		Data:    map[string]string{"title": "test"},
	}

	hub.Broadcast("board1", evt)

	select {
	case msg := <-sendCh:
		var received Event
		if err := json.Unmarshal(msg, &received); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if received.Type != EventTaskCreated {
			t.Errorf("expected type=%s, got %s", EventTaskCreated, received.Type)
		}
		if received.SequenceID != 1 {
			t.Errorf("expected sequence_id=1, got %d", received.SequenceID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}

	hub.Unregister(client)
}

func TestHubReplayMissedEvents(t *testing.T) {
	fr := newFakeRedis()
	fr.mu.Lock()
	fr.history["board:board1:history"] = []string{
		`{"sequence_id":1,"board_id":"board1","type":"task.created","actor_id":"user1","data":{"title":"old"}}`,
		`{"sequence_id":2,"board_id":"board1","type":"task.updated","actor_id":"user1","data":{"title":"new"}}`,
	}
	fr.mu.Unlock()

	hub := NewHub(fr)

	sendCh := make(chan []byte, 256)
	client := &Client{
		hub:     hub,
		userID:  "user2",
		boardID: "board1",
		lastSeq: 1,
		send:    sendCh,
		done:    make(chan struct{}),
	}

	hub.Register(client)
	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-sendCh:
		var received Event
		if err := json.Unmarshal(msg, &received); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if received.SequenceID != 2 {
			t.Errorf("expected replayed sequence_id=2, got %d", received.SequenceID)
		}
	default:
		t.Fatal("expected replayed messages")
	}

	hub.Unregister(client)
}
