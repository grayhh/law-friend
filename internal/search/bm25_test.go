package search

import "testing"

func TestTokenize_Korean(t *testing.T) {
	got := Tokenize("전세 보증금 반환")
	want := map[string]bool{
		"전세": true, "보증금": true, "반환": true,
		"보증": true, "증금": true,
	}
	for _, tok := range got {
		delete(want, tok)
	}
	if len(want) > 0 {
		t.Fatalf("missing tokens: %v (got=%v)", want, got)
	}
}

func TestBM25_RankRelevant(t *testing.T) {
	docs := []Document{
		{ID: "a", Text: "임대차 보증금 반환 청구 사건"},
		{ID: "b", Text: "교통사고 손해배상 청구"},
		{ID: "c", Text: "이혼 위자료 양육권 분쟁"},
		{ID: "d", Text: "보증금 미반환 임대인 채무불이행"},
	}
	idx := NewIndex(docs)

	hits := idx.Search("전세 보증금 못 받음", 3)
	if len(hits) == 0 {
		t.Fatal("expected at least 1 hit, got 0")
	}
	// "a" 또는 "d"가 상위에 와야 함 (보증금 관련)
	top := hits[0].ID
	if top != "a" && top != "d" {
		t.Errorf("expected top hit a or d, got %s (hits=%v)", top, hits)
	}
	// "c"(이혼)는 매칭되면 안 됨
	for _, h := range hits {
		if h.ID == "c" {
			t.Errorf("non-relevant doc c should not match, got hits=%v", hits)
		}
	}
}

func TestBM25_NoMatch(t *testing.T) {
	docs := []Document{
		{ID: "a", Text: "임대차 보증금"},
	}
	idx := NewIndex(docs)
	hits := idx.Search("주식 양도소득세", 3)
	if len(hits) != 0 {
		t.Errorf("expected no hits, got %v", hits)
	}
}
