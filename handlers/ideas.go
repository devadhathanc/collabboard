package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"ideaboard/board"
	"ideaboard/ws"
)

type ideasHandler struct {
	store *board.Store
	hub   *ws.Hub
}

// NewIdeasHandler creates the handler.
func NewIdeasHandler(store *board.Store, hub *ws.Hub) *ideasHandler {
	return &ideasHandler{store: store, hub: hub}
}

// GET /api/boards/{boardID}/ideas
func (h *ideasHandler) List(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("boardID")
	b, ok := h.store.Get(boardID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ideas": []interface{}{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ideas": b.ListIdeas()})
}

// POST /api/boards/{boardID}/ideas
func (h *ideasHandler) Submit(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("boardID")

	var req struct {
		Text      string `json:"text"`
		CreatedBy string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if len(req.Text) < 1 {
		http.Error(w, `{"error":"text required"}`, http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "anonymous"
	}

	b := h.store.GetOrCreate(boardID)
	idea, isNew := b.SubmitIdea(req.Text, req.CreatedBy)

	msgType := ws.MsgIdeaAdded
	if !isNew {
		msgType = ws.MsgIdeaUpdated
	}
	h.hub.Broadcast(boardID, msgType, idea)

	writeJSON(w, http.StatusCreated, idea)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
