package board

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/devadhathan/collabboard/internal/auth"
	"github.com/devadhathan/collabboard/internal/db"
	"github.com/devadhathan/collabboard/internal/ws"
)

type Handler struct {
	svc *Service
	hub *ws.Hub
}

func NewHandler(svc *Service, hub *ws.Hub) *Handler {
	return &Handler{svc: svc, hub: hub}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /boards/{boardID}/tasks", h.CreateTask)
	mux.HandleFunc("PATCH /boards/{boardID}/tasks/{taskID}", h.UpdateTask)
	mux.HandleFunc("PATCH /boards/{boardID}/tasks", h.BulkUpdateTasks)
	mux.HandleFunc("GET /boards/{boardID}/tasks/{taskID}", h.GetTask)
}

func userIDFromContext(r *http.Request) string {
	claims := auth.GetClaims(r.Context())
	if claims != nil {
		return claims.UserID
	}
	return "unknown"
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

	if h.hub != nil {
		h.hub.PublishAndBroadcast(r.Context(), boardID, &ws.Event{
			BoardID: boardID,
			Type:    ws.EventTaskCreated,
			ActorID: userIDFromContext(r),
			Data:    task,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

type updateTaskRequest struct {
	Title      *string  `json:"title,omitempty"`
	Status     *string  `json:"status,omitempty"`
	AssigneeID *string  `json:"assignee_id,omitempty"`
	Position   *float64 `json:"position,omitempty"`
	Version    int      `json:"version"`
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	boardID := r.PathValue("boardID")

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	current, err := h.svc.store.GetByID(r.Context(), taskID)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	if req.Version > 0 && current.Version != req.Version {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "version conflict",
			"current": current,
		})
		return
	}

	if req.Title != nil {
		current.Title = *req.Title
	}
	if req.Status != nil {
		current.Status = *req.Status
	}
	if req.AssigneeID != nil {
		current.AssigneeID = req.AssigneeID
	}
	if req.Position != nil {
		current.Position = *req.Position
	}

	ok, err := h.svc.store.UpdateWithVersion(r.Context(), current, current.Version)
	if err != nil {
		log.Printf("update task error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		latest, _ := h.svc.store.GetByID(r.Context(), taskID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "version conflict",
			"current": latest,
		})
		return
	}

	if h.hub != nil {
		h.hub.PublishAndBroadcast(r.Context(), boardID, &ws.Event{
			BoardID: boardID,
			Type:    ws.EventTaskUpdated,
			ActorID: userIDFromContext(r),
			Data:    current,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(current)
}

type bulkUpdateItem struct {
	ID      string  `json:"id"`
	Title   *string `json:"title,omitempty"`
	Status  *string `json:"status,omitempty"`
	Position float64 `json:"position"`
	Version int     `json:"version"`
}

func (h *Handler) BulkUpdateTasks(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("boardID")

	var req struct {
		Tasks []bulkUpdateItem `json:"tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	type conflict struct {
		ID      string   `json:"id"`
		Current *db.Task `json:"current"`
	}
	var conflicts []conflict
	var updated []*db.Task

	for _, item := range req.Tasks {
		current, err := h.svc.store.GetByID(r.Context(), item.ID)
		if err != nil {
			continue
		}

		if current.Version != item.Version {
			conflicts = append(conflicts, conflict{ID: item.ID, Current: current})
			continue
		}

		if item.Title != nil {
			current.Title = *item.Title
		}
		if item.Status != nil {
			current.Status = *item.Status
		}
		current.Position = item.Position

		ok, err := h.svc.store.UpdateWithVersion(r.Context(), current, item.Version)
		if err != nil || !ok {
			latest, _ := h.svc.store.GetByID(r.Context(), item.ID)
			conflicts = append(conflicts, conflict{ID: item.ID, Current: latest})
			continue
		}
		updated = append(updated, current)
	}

	if h.hub != nil {
		for _, t := range updated {
			h.hub.PublishAndBroadcast(r.Context(), boardID, &ws.Event{
				BoardID: boardID,
				Type:    ws.EventTaskMoved,
				ActorID: userIDFromContext(r),
				Data:    t,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(conflicts) > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updated":   updated,
			"conflicts": conflicts,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"updated": updated,
	})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")

	task, err := h.svc.store.GetByID(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, errors.New("not found")) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
