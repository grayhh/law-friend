// Package search provides simple in-memory lexical search.
//
// BM25 over a Korean+ASCII tokenizer: whitespace+punctuation splitting,
// ASCII lowercasing, and character bigrams for CJK runs. No external deps.
package search

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type Document struct {
	ID   string
	Text string
}

type docEntry struct {
	id     string
	counts map[string]int
	length int
}

type Index struct {
	docs  []docEntry
	df    map[string]int
	avgdl float64
	k1, b float64
}

func NewIndex(docs []Document) *Index {
	idx := &Index{
		df: make(map[string]int),
		k1: 1.5,
		b:  0.75,
	}
	totalLen := 0
	for _, d := range docs {
		toks := Tokenize(d.Text)
		counts := make(map[string]int, len(toks))
		for _, t := range toks {
			counts[t]++
		}
		for t := range counts {
			idx.df[t]++
		}
		idx.docs = append(idx.docs, docEntry{
			id:     d.ID,
			counts: counts,
			length: len(toks),
		})
		totalLen += len(toks)
	}
	if len(idx.docs) > 0 {
		idx.avgdl = float64(totalLen) / float64(len(idx.docs))
	}
	return idx
}

type Hit struct {
	ID    string
	Score float64
}

func (idx *Index) Search(query string, topK int) []Hit {
	if idx == nil || len(idx.docs) == 0 {
		return nil
	}
	qTokens := Tokenize(query)
	if len(qTokens) == 0 {
		return nil
	}
	// dedupe query tokens to avoid double-counting
	seen := make(map[string]struct{}, len(qTokens))
	uniq := qTokens[:0]
	for _, t := range qTokens {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}

	N := float64(len(idx.docs))
	hits := make([]Hit, 0, len(idx.docs))
	for _, d := range idx.docs {
		var score float64
		for _, qt := range uniq {
			f := float64(d.counts[qt])
			if f == 0 {
				continue
			}
			df := float64(idx.df[qt])
			idf := math.Log((N-df+0.5)/(df+0.5) + 1)
			denom := f + idx.k1*(1-idx.b+idx.b*float64(d.length)/idx.avgdl)
			score += idf * (f * (idx.k1 + 1)) / denom
		}
		if score > 0 {
			hits = append(hits, Hit{ID: d.id, Score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if topK > 0 && topK < len(hits) {
		hits = hits[:topK]
	}
	return hits
}

// Tokenize 분리·소문자화하고, CJK 단어는 추가로 문자 bigram을 발생시킴.
//
// 입력 "전세 보증금"
//
//	-> 단어: "전세", "보증금"
//	-> bigram: "전세", "보증", "증금"
//	-> 최종(중복 포함): 전세, 전세, 보증금, 보증, 증금
func Tokenize(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ToLower(s)
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var out []string
	for _, f := range fields {
		if f == "" {
			continue
		}
		out = append(out, f)
		runes := []rune(f)
		if len(runes) >= 2 && hasCJK(runes) {
			for i := 0; i < len(runes)-1; i++ {
				out = append(out, string(runes[i:i+2]))
			}
		}
	}
	return out
}

func hasCJK(rs []rune) bool {
	for _, r := range rs {
		// Hangul Syllables, Jamo, CJK Unified Ideographs (rough but enough for our use).
		if (r >= 0xAC00 && r <= 0xD7A3) ||
			(r >= 0x1100 && r <= 0x11FF) ||
			(r >= 0x3130 && r <= 0x318F) ||
			(r >= 0x4E00 && r <= 0x9FFF) {
			return true
		}
	}
	return false
}
