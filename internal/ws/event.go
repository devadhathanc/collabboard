package ws

import "time"

type EventType string

const (
	EventTaskCreated    EventType = "task.created"
	EventTaskUpdated    EventType = "task.updated"
	EventTaskDeleted    EventType = "task.deleted"
	EventTaskMoved      EventType = "task.moved"
	EventBoardUpdated   EventType = "board.updated"
	EventCursorMove     EventType = "cursor.move"
	EventResync         EventType = "resync"
)

type Event struct {
	ID         string      `json:"id"`
	SequenceID int64       `json:"sequence_id"`
	BoardID    string      `json:"board_id"`
	Type       EventType   `json:"type"`
	ActorID    string      `json:"actor_id"`
	Data       interface{} `json:"data"`
	Timestamp  time.Time   `json:"timestamp"`
}

type CursorData struct {
	UserID string  `json:"user_id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}
