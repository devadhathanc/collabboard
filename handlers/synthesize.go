package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"ideaboard/board"
	"ideaboard/ollama"
	"ideaboard/ws"
)

type SynthesizeHandler struct {
	store  *board.Store
	ollama *ollama.Client
	hub    *ws.Hub
}

func NewSynthesizeHandler(store *board.Store, ollama *ollama.Client, hub *ws.Hub) *SynthesizeHandler {
	return &SynthesizeHandler{store: store, ollama: ollama, hub: hub}
}

func (h *SynthesizeHandler) Synthesize(w http.ResponseWriter, r *http.Request) {
	boardID := r.PathValue("boardID")

	if !h.ollama.Enabled() {
		http.Error(w, "Ollama not running — start with: ollama serve", http.StatusNotImplemented)
		return
	}

	// Get or create the board and collect existing idea texts.
	b := h.store.GetOrCreate(boardID)
	existing := b.ListIdeas()

	if len(existing) == 0 {
		http.Error(w, "no ideas on this board yet", http.StatusBadRequest)
		return
	}

	texts := make([]string, 0, len(existing))
	for _, idea := range existing {
		texts = append(texts, idea.Text)
	}

	// Ask Gemini to generate new related ideas.
	result, err := h.ollama.Synthesize(r.Context(), texts)
	if err != nil {
		log.Printf("[synthesize] ollama error: %v", err)
		http.Error(w, "AI brainstorm failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Hard-cap at 3 ideas regardless of what the model returns.
	generatedIdeas := result.Ideas
	if len(generatedIdeas) > 3 {
		generatedIdeas = generatedIdeas[:3]
	}

	// Submit each generated idea to the board as "AI" and broadcast via WS.
	added := make([]*board.Idea, 0, len(generatedIdeas))
	for _, text := range generatedIdeas {
		if text == "" {
			continue
		}
		idea, isNew := b.SubmitIdea(text, "AI")
		if isNew {
			h.hub.Broadcast(boardID, ws.MsgIdeaAdded, idea)
		} else {
			h.hub.Broadcast(boardID, ws.MsgIdeaUpdated, idea)
		}
		added = append(added, idea)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"added": len(added),
		"ideas": added,
	})
}
