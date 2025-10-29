package rag

import (
	"regexp"
	"strings"
)

// Tokenizer provides lightweight tokenisation suitable for deterministic local embeddings.
// It operates entirely offline and keeps allocations predictable; the implementation deliberately
// avoids any external model files so that we can run in locked-down desktop environments.
type Tokenizer struct {
	wordPattern *regexp.Regexp
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		wordPattern: regexp.MustCompile(`[[:alnum:]_]+`),
	}
}

// Tokenize returns placeholder token ids / attention masks sized to input length.
// This keeps compatibility with any callers that still expect "transformer style" ids.
func (t *Tokenizer) Tokenize(text string) ([]int64, []int64) {
	n := len(text)
	if n == 0 {
		return []int64{101, 102}, []int64{1, 1} // [CLS][SEP]
	}
	if n > 256 {
		n = 256
	}
	ids := make([]int64, n)
	mask := make([]int64, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(100 + (i % 97))
		mask[i] = 1
	}
	return ids, mask
}

// Terms extracts normalised lexical units that are fed into the local embedding projector.
// The goal is to approximate the effect of a WordPiece/BPE tokenizer without needing model assets.
func (t *Tokenizer) Terms(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	lower := strings.ToLower(text)
	raw := t.wordPattern.FindAllString(lower, -1)
	if len(raw) == 0 {
		return []string{lower}
	}

	terms := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		terms = append(terms, token)
	}
	return terms
}
