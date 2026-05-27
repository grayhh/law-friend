package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("session not found")

type fileLayout struct {
	Sessions []*Session `json:"sessions"`
}

type Store struct {
	path string
	mu   sync.Mutex
	data fileLayout
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".lawmate", "sessions.json"), nil
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("ensure dir: %w", err)
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = fileLayout{Sessions: []*Session{}}
			return nil
		}
		return fmt.Errorf("read sessions file: %w", err)
	}
	if len(b) == 0 {
		s.data = fileLayout{Sessions: []*Session{}}
		return nil
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return fmt.Errorf("parse sessions file: %w", err)
	}
	if s.data.Sessions == nil {
		s.data.Sessions = []*Session{}
	}
	return nil
}

func (s *Store) persistLocked() error {
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".sessions-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

func (s *Store) List() []Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Summary, 0, len(s.data.Sessions))
	for _, sess := range s.data.Sessions {
		out = append(out, sess.Summary())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (s *Store) Get(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.data.Sessions {
		if sess.ID == id {
			return cloneSession(sess), nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sess := range s.data.Sessions {
		if sess.ID == id {
			s.data.Sessions = append(s.data.Sessions[:i], s.data.Sessions[i+1:]...)
			return s.persistLocked()
		}
	}
	return ErrNotFound
}

// Upsert creates a new session if id is empty, otherwise appends to existing.
// Returns the updated session (clone).
func (s *Store) Upsert(id, cli string, userMsg Message, assistantMsg Message) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	if id == "" {
		newID := generateID()
		sess := &Session{
			ID:        newID,
			Title:     makeTitle(userMsg.Content),
			CLI:       cli,
			CreatedAt: now,
			UpdatedAt: now,
			Messages:  []Message{userMsg, assistantMsg},
		}
		s.data.Sessions = append(s.data.Sessions, sess)
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
		return cloneSession(sess), nil
	}

	for _, sess := range s.data.Sessions {
		if sess.ID == id {
			sess.Messages = append(sess.Messages, userMsg, assistantMsg)
			sess.UpdatedAt = now
			if err := s.persistLocked(); err != nil {
				return nil, err
			}
			return cloneSession(sess), nil
		}
	}
	return nil, ErrNotFound
}

func cloneSession(s *Session) *Session {
	cp := *s
	cp.Messages = append([]Message(nil), s.Messages...)
	return &cp
}

func makeTitle(text string) string {
	runes := []rune(text)
	const max = 30
	if len(runes) > max {
		return string(runes[:max]) + "…"
	}
	return text
}

func generateID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("ses_%s_%s", time.Now().Format("20060102_150405"), hex.EncodeToString(b[:]))
}
