package board

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"
)

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Idea represents a single submitted idea.
type Idea struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"board_id"`
	Text      string    `json:"text"`
	Count     int       `json:"count"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Board holds all ideas for a session, protected by a mutex.
type Board struct {
	ID    string
	Ideas map[string]*Idea // keyed by idea ID
	mu    sync.RWMutex
}

// Store holds all boards in memory.
type Store struct {
	mu     sync.RWMutex
	boards map[string]*Board
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
	return &Store{boards: make(map[string]*Board)}
}

// GetOrCreate returns an existing board or creates a new one.
func (s *Store) GetOrCreate(boardID string) *Board {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.boards[boardID]; ok {
		return b
	}
	b := &Board{ID: boardID, Ideas: make(map[string]*Idea)}
	s.boards[boardID] = b
	return b
}

// Get returns a board if it exists.
func (s *Store) Get(boardID string) (*Board, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.boards[boardID]
	return b, ok
}

// SubmitIdea adds a new idea or increments the count if an exact match exists.
// Returns the idea and whether it was newly created.
func (b *Board) SubmitIdea(text, createdBy string) (*Idea, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	normalised := strings.ToLower(strings.TrimSpace(text))

	// Look for an existing idea with the same normalised text.
	for _, idea := range b.Ideas {
		if strings.ToLower(strings.TrimSpace(idea.Text)) == normalised {
			idea.Count++
			return idea, false // existing, count bumped
		}
	}

	// New idea.
	idea := &Idea{
		ID:        newID(),
		BoardID:   b.ID,
		Text:      strings.TrimSpace(text),
		Count:     1,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
	b.Ideas[idea.ID] = idea
	return idea, true
}

// ListIdeas returns all ideas for a board, sorted newest-first.
func (b *Board) ListIdeas() []*Idea {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]*Idea, 0, len(b.Ideas))
	for _, idea := range b.Ideas {
		out = append(out, idea)
	}
	// Sort by created_at descending.
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// IdeaCount returns the number of ideas on a board.
func (b *Board) IdeaCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.Ideas)
}
