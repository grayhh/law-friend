package precedents

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chan/lawmate/internal/search"
)

type DummyStore struct {
	items []Precedent
	index map[string]*Precedent
	bm25  *search.Index
}

func LoadDummyStore(path string) (*DummyStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read precedents file: %w", err)
	}
	return parseStore(data)
}

func LoadDummyStoreFromBytes(data []byte) (*DummyStore, error) {
	return parseStore(data)
}

func parseStore(data []byte) (*DummyStore, error) {
	var items []Precedent
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse precedents json: %w", err)
	}
	index := make(map[string]*Precedent, len(items))
	docs := make([]search.Document, len(items))
	for i := range items {
		index[items[i].ID] = &items[i]
		docs[i] = search.Document{
			ID:   items[i].ID,
			Text: indexableText(&items[i]),
		}
	}
	return &DummyStore{
		items: items,
		index: index,
		bm25:  search.NewIndex(docs),
	}, nil
}

// indexableText: 핵심 필드를 가중 반복으로 포함.
// case_name·issues·summary는 핵심 검색 신호라 2회씩.
// full_text는 노이즈가 많아 1회만.
func indexableText(p *Precedent) string {
	parts := []string{
		p.CaseName, p.CaseName,
		p.Issues, p.Issues,
		p.Summary, p.Summary,
		p.ReferencesLaw,
		p.FullText,
	}
	return strings.Join(parts, "\n")
}

func (s *DummyStore) Search(query string, k int) ([]Precedent, error) {
	if len(s.items) == 0 {
		return nil, nil
	}
	hits := s.bm25.Search(query, k)
	if len(hits) == 0 {
		// fallback: 검색이 비면 가장 최신 K건 정도라도 반환하지 말고
		// 빈 결과를 돌려준다 — 프롬프트가 "관련 판례 없음"을 솔직하게 표현.
		return nil, nil
	}
	out := make([]Precedent, 0, len(hits))
	for _, h := range hits {
		if p, ok := s.index[h.ID]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (s *DummyStore) GetByID(id string) (*Precedent, error) {
	p, ok := s.index[id]
	if !ok {
		return nil, fmt.Errorf("precedent not found: %s", id)
	}
	return p, nil
}

func (s *DummyStore) All() []Precedent {
	return s.items
}
