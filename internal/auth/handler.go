package auth

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("POST /auth/revoke", h.Revoke)
	mux.HandleFunc("GET  /.well-known/jwks.json", h.JWKS)
}

type loginRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Password == "" {
		http.Error(w, `{"error":"user_id and password required"}`, http.StatusBadRequest)
		return
	}

	accessToken, err := h.svc.IssueAccessToken(req.UserID, nil)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	refreshToken, _, err := h.svc.IssueRefreshToken(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.svc.accessTTL.Seconds()),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	newAccess, newRefresh, err := h.svc.RotateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		ExpiresIn:    int(h.svc.accessTTL.Seconds()),
	})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*Claims)
	if err := h.svc.denylist.Add(r.Context(), claims.ID, h.svc.accessTTL); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	data, _ := JSONJWKS(h.svc)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
