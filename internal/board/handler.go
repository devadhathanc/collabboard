package board

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/devadhathan/collabboard/internal/db"
	"github.com/devadhathan/collabboard/internal/redisc"
	"github.com/devadhathan/collabboard/internal/ws"
)

type Handler struct {
	svc *Service
	hub *ws.Hub
	rc  *redisc.Client
}

func NewHandler(svc *Service, hub *ws.Hub, rc *redisc.Client) *Handler {
	return &Handler{svc: svc, hub: hub, rc: rc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /boards/{boardID}/tasks", h.CreateTask)
	mux.HandleFunc("GET /boards/{boardID}/tasks/{taskID}", h.GetTask)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title      string  `json:"title"`
		AssigneeID *string `json:"assignee_id"`
		Position   float64 `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	boardID := r.PathValue("boardID")

	task := &db.Task{
		BoardID:    boardID,
		Title:      req.Title,
		Status:     "todo",
		AssigneeID: req.AssigneeID,
		Position:   req.Position,
	}

	if err := h.svc.store.Create(r.Context(), task); err != nil {
		log.Printf("create task error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	h.hub.PublishAndBroadcast(r.Context(), boardID, &ws.Event{
		BoardID: boardID,
		Type:    ws.EventTaskCreated,
		ActorID: r.Context().Value("user_id").(string),
		Data:    task,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")

	task, err := h.svc.store.GetByID(r.Context(), taskID)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
