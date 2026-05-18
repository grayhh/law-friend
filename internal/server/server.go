package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/chan/beopchin/internal/llm"
	"github.com/chan/beopchin/internal/precedents"
	"github.com/chan/beopchin/internal/prompt"
	"github.com/chan/beopchin/internal/sessions"
)

type Server struct {
	store    precedents.Store
	sessions *sessions.Store
	webFS    fs.FS
	topK     int
	llmTimeo time.Duration
}

func New(store precedents.Store, sessStore *sessions.Store, webFS fs.FS) *Server {
	return &Server{
		store:    store,
		sessions: sessStore,
		webFS:    webFS,
		topK:     3,
		llmTimeo: 120 * time.Second,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/chat", s.handleChat)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/precedents/{id}", s.handleGetPrecedent)
	mux.HandleFunc("GET /api/cli/available", s.handleCLIAvailable)
	mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", s.handleDeleteSession)
	mux.Handle("/", http.FileServerFS(s.webFS))
	return mux
}

type chatRequest struct {
	Query     string           `json:"query"`
	History   []prompt.Message `json:"history"`
	CLI       string           `json:"cli"`
	SessionID string           `json:"session_id,omitempty"`
}

type chatResponse struct {
	Answer     string                 `json:"answer"`
	Precedents []precedents.Precedent `json:"precedents"`
	CLI        string                 `json:"cli"`
	SessionID  string                 `json:"session_id"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}
	if req.CLI == "" {
		http.Error(w, "cli is required (claude|codex)", http.StatusBadRequest)
		return
	}

	// If continuing an existing session, enforce its CLI (read-only sessions can't switch CLI).
	if req.SessionID != "" {
		existing, err := s.sessions.Get(req.SessionID)
		if err == nil && existing.CLI != req.CLI {
			http.Error(w, "session is bound to a different CLI ("+existing.CLI+")", http.StatusBadRequest)
			return
		}
	}

	client, err := llm.New(llm.CLIKind(req.CLI))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hits, err := s.store.Search(req.Query, s.topK)
	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	p := prompt.Build(req.History, req.Query, hits)

	ctx, cancel := context.WithTimeout(r.Context(), s.llmTimeo)
	defer cancel()
	answer, err := client.Ask(ctx, p)
	if err != nil {
		http.Error(w, "llm failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	userMsg := sessions.Message{Role: "user", Content: req.Query}
	assistantMsg := sessions.Message{Role: "assistant", Content: answer, Precedents: hits}

	sess, err := s.sessions.Upsert(req.SessionID, req.CLI, userMsg, assistantMsg)
	if err != nil {
		http.Error(w, "session save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, chatResponse{
		Answer:     answer,
		Precedents: hits,
		CLI:        req.CLI,
		SessionID:  sess.ID,
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q (query) is required", http.StatusBadRequest)
		return
	}
	k := s.topK
	if kp := r.URL.Query().Get("k"); kp != "" {
		if v, err := strconv.Atoi(kp); err == nil && v > 0 {
			k = v
		}
	}
	hits, err := s.store.Search(q, k)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":      q,
		"k":          k,
		"precedents": hits,
	})
}

func (s *Server) handleGetPrecedent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleCLIAvailable(w http.ResponseWriter, _ *http.Request) {
	available := llm.Detect()
	out := make([]string, 0, len(available))
	for _, k := range available {
		out = append(out, string(k))
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": out})
}

func (s *Server) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"sessions": s.sessions.List()})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.sessions.Get(id)
	if err != nil {
		if errors.Is(err, sessions.ErrNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.sessions.Delete(id); err != nil {
		if errors.Is(err, sessions.ErrNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func MustSub(efs embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(efs, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
