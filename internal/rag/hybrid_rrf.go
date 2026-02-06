package rag

import (
	"context"
	"fmt"
	"sort"
)

// HybridSearch combines vector and text search using Reciprocal Rank Fusion (RRF)
func (s *SQLiteVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
	// Fetch more candidates for re-ranking (3x final result count)
	candidateCount := k * 3

	// Execute both searches in parallel
	type searchResult struct {
		docs []*Document
		err  error
	}

	vectorChan := make(chan searchResult, 1)
	textChan := make(chan searchResult, 1)

	// Vector search
	go func() {
		docs, err := s.SearchSimilar(ctx, embedding, candidateCount, nil)
		vectorChan <- searchResult{docs: docs, err: err}
	}()

	// Text search
	go func() {
		docs, err := s.SearchByText(ctx, query, candidateCount, nil)
		textChan <- searchResult{docs: docs, err: err}
	}()

	// Collect results
	var vectorResults, textResults []*Document
	for i := 0; i < 2; i++ {
		select {
		case result := <-vectorChan:
			if result.err != nil {
				return nil, fmt.Errorf("vector search failed: %w", result.err)
			}
			vectorResults = result.docs
		case result := <-textChan:
			if result.err != nil {
				s.logger.WithError(result.err).Warn("Text search failed, using vector results only")
				textResults = []*Document{}
			} else {
				textResults = result.docs
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Apply RRF fusion
	return s.fuseWithRRF(vectorResults, textResults, k)
}

// fuseWithRRF applies Reciprocal Rank Fusion to combine vector and text search results
func (s *SQLiteVectorStore) fuseWithRRF(vectorResults []*Document, textResults []*Document, k int) ([]*Document, error) {
	// Track RRF scores and document map
	rrfScores := make(map[string]float64)
	docMap := make(map[string]*Document)

	// Track ranks for debugging/transparency
	vectorRanks := make(map[string]int)
	textRanks := make(map[string]int)

	// Process vector results
	for rank, doc := range vectorResults {
		docID := doc.ID
		vectorRanks[docID] = rank

		// RRF score = weight / (rank + k)
		score := s.vectorWeight / float64(rank+1+s.rrfConstant)
		rrfScores[docID] = score

		// Store document
		if _, exists := docMap[docID]; !exists {
			docMap[docID] = doc
		}
	}

	// Process text results
	for rank, doc := range textResults {
		docID := doc.ID
		textRanks[docID] = rank

		// Add to existing RRF score or create new
		score := s.textWeight / float64(rank+1+s.rrfConstant)
		rrfScores[docID] += score

		// Store document if not already present
		if _, exists := docMap[docID]; !exists {
			docMap[docID] = doc
		}
	}

	// Create sorted result list
	type scoredDoc struct {
		doc   *Document
		score float64
	}

	scored := make([]scoredDoc, 0, len(rrfScores))
	for docID, score := range rrfScores {
		doc := docMap[docID]
		doc.Score = float32(score)

		// Add ranking metadata for transparency
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}

		if vRank, hasVector := vectorRanks[docID]; hasVector {
			doc.Metadata["vector_rank"] = vRank + 1
		}
		if tRank, hasText := textRanks[docID]; hasText {
			doc.Metadata["text_rank"] = tRank + 1
		}
		doc.Metadata["rrf_score"] = score

		scored = append(scored, scoredDoc{doc: doc, score: score})
	}

	// Sort by RRF score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top-k
	results := make([]*Document, 0, k)
	for i := 0; i < len(scored) && i < k; i++ {
		results = append(results, scored[i].doc)
	}

	return results, nil
}
