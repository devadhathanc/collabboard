package board

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/devadhathan/collabboard/internal/db"
	"github.com/devadhathan/collabboard/internal/ws"
)

type mockStore struct {
	mu    sync.Mutex
	tasks map[string]*db.Task
}

func newMockStore() *mockStore {
	return &mockStore{tasks: make(map[string]*db.Task)}
}

func (s *mockStore) Create(_ context.Context, t *db.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.ID = "task-" + t.Title
	t.Version = 1
	s.tasks[t.ID] = t
	return nil
}

func (s *mockStore) GetByID(_ context.Context, id string) (*db.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (s *mockStore) UpdateWithVersion(_ context.Context, t *db.Task, expectedVersion int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tasks[t.ID]
	if !ok || existing.Version != expectedVersion {
		return false, nil
	}
	existing.Version++
	existing.Title = t.Title
	existing.Status = t.Status
	existing.AssigneeID = t.AssigneeID
	existing.Position = t.Position
	t.Version = existing.Version
	return true, nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }

type mockHub struct{}

func (m *mockHub) PublishAndBroadcast(_ context.Context, _ string, _ *ws.Event) {}

func TestOCCUpdateSuccess(t *testing.T) {
	store := newMockStore()
	store.tasks["t1"] = &db.Task{ID: "t1", Title: "hello", Version: 1, Status: "todo"}

	h := &Handler{
		svc: &Service{store: store, pub: &mockPublisher{}},
	}

	body := `{"title":"updated","status":"done","version":1}`
	req := httptest.NewRequest("PATCH", "/boards/b1/tasks/t1", strings.NewReader(body))
	req.SetPathValue("boardID", "b1")
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.UpdateTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp db.Task
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Title != "updated" {
		t.Errorf("expected title=updated, got %s", resp.Title)
	}
	if resp.Version != 2 {
		t.Errorf("expected version=2, got %d", resp.Version)
	}
}

func TestOCCConflict(t *testing.T) {
	store := newMockStore()
	store.tasks["t1"] = &db.Task{ID: "t1", Title: "hello", Version: 1, Status: "todo"}

	h := &Handler{
		svc: &Service{store: store, pub: &mockPublisher{}},
	}

	body := `{"title":"updated","version":2}`
	req := httptest.NewRequest("PATCH", "/boards/b1/tasks/t1", strings.NewReader(body))
	req.SetPathValue("boardID", "b1")
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.UpdateTask(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "version conflict" {
		t.Errorf("expected version conflict error, got %v", resp["error"])
	}
}

func TestOCCBulkPartialConflict(t *testing.T) {
	store := newMockStore()
	store.tasks["t1"] = &db.Task{ID: "t1", Title: "a", Version: 1, Status: "todo"}
	store.tasks["t2"] = &db.Task{ID: "t2", Title: "b", Version: 1, Status: "todo"}

	h := &Handler{
		svc: &Service{store: store, pub: &mockPublisher{}},
	}

	body := `{"tasks":[
		{"id":"t1","title":"a-updated","position":1,"version":1},
		{"id":"t2","title":"b-updated","position":2,"version":99}
	]}`
	req := httptest.NewRequest("PATCH", "/boards/b1/tasks", strings.NewReader(body))
	req.SetPathValue("boardID", "b1")
	w := httptest.NewRecorder()

	h.BulkUpdateTasks(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for partial conflict, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	conflicts := resp["conflicts"].([]interface{})
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
}
