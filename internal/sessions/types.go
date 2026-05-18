package sessions

import (
	"time"

	"github.com/chan/beopchin/internal/precedents"
)

type Message struct {
	Role       string                 `json:"role"` // "user" | "assistant"
	Content    string                 `json:"content"`
	Precedents []precedents.Precedent `json:"precedents,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CLI       string    `json:"cli"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
}

type Summary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CLI       string    `json:"cli"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Session) Summary() Summary {
	return Summary{
		ID:        s.ID,
		Title:     s.Title,
		CLI:       s.CLI,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
