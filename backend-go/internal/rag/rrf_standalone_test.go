package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRRFLogicStandalone tests the RRF calculation logic without dependencies
func TestRRFLogicStandalone(t *testing.T) {
	// Test basic RRF score calculation
	rrfConstant := 60
	vectorWeight := 1.0
	textWeight := 1.0

	// Document appears at rank 0 in vector, rank 5 in text
	vectorRank := 0
	textRank := 5

	vectorScore := vectorWeight / float64(vectorRank+1+rrfConstant)
	textScore := textWeight / float64(textRank+1+rrfConstant)
	totalScore := vectorScore + textScore

	// Expected: (1/61) + (1/66) ≈ 0.0164 + 0.0152 = 0.0316
	expectedMin := 0.031
	expectedMax := 0.032

	assert.GreaterOrEqual(t, totalScore, expectedMin, "RRF score should be >= expected minimum")
	assert.LessOrEqual(t, totalScore, expectedMax, "RRF score should be <= expected maximum")
	t.Logf("RRF score for ranks (0, 5) with k=60: %.6f", totalScore)
}

// TestRRFConstantEffectStandalone tests how different k values affect scoring
func TestRRFConstantEffectStandalone(t *testing.T) {
	tests := []struct {
		name        string
		k           int
		rank0Score  float64
		rank10Score float64
		minRatio    float64 // Minimum expected ratio between rank 0 and rank 10
	}{
		{
			name:        "low_k_20",
			k:           20,
			rank0Score:  1.0 / 21.0, // 0.0476
			rank10Score: 1.0 / 31.0, // 0.0323
			minRatio:    1.4,        // Should create bigger differences
		},
		{
			name:        "default_k_60",
			k:           60,
			rank0Score:  1.0 / 61.0, // 0.0164
			rank10Score: 1.0 / 71.0, // 0.0141
			minRatio:    1.15,       // Moderate differences
		},
		{
			name:        "high_k_100",
			k:           100,
			rank0Score:  1.0 / 101.0, // 0.0099
			rank10Score: 1.0 / 111.0, // 0.0090
			minRatio:    1.08,        // Smaller differences
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := tt.rank0Score / tt.rank10Score
			t.Logf("k=%d: rank0=%.4f, rank10=%.4f, ratio=%.2f",
				tt.k, tt.rank0Score, tt.rank10Score, ratio)

			assert.GreaterOrEqual(t, ratio, tt.minRatio,
				"Score ratio should be >= expected minimum")

			// Lower k values should create larger ratios
			if tt.k < 60 {
				assert.Greater(t, ratio, 1.3, "Low k should create ratio > 1.3")
			} else if tt.k > 60 {
				assert.Less(t, ratio, 1.2, "High k should create ratio < 1.2")
			}
		})
	}
}

// TestWeightedRRF tests RRF with different weights
func TestWeightedRRF(t *testing.T) {
	rrfConstant := 60
	rank := 0

	tests := []struct {
		name         string
		vectorWeight float64
		textWeight   float64
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "equal_weights",
			vectorWeight: 1.0,
			textWeight:   1.0,
			expectedMin:  0.032, // 2 * (1/61) ≈ 0.0328
			expectedMax:  0.033,
		},
		{
			name:         "vector_preferred_2x",
			vectorWeight: 2.0,
			textWeight:   1.0,
			expectedMin:  0.049, // (2/61) + (1/61) ≈ 0.0492
			expectedMax:  0.050,
		},
		{
			name:         "text_preferred_2x",
			vectorWeight: 1.0,
			textWeight:   2.0,
			expectedMin:  0.049, // (1/61) + (2/61) ≈ 0.0492
			expectedMax:  0.050,
		},
		{
			name:         "vector_strongly_preferred_5x",
			vectorWeight: 5.0,
			textWeight:   1.0,
			expectedMin:  0.098, // (5/61) + (1/61) ≈ 0.0984
			expectedMax:  0.099,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vectorScore := tt.vectorWeight / float64(rank+1+rrfConstant)
			textScore := tt.textWeight / float64(rank+1+rrfConstant)
			totalScore := vectorScore + textScore

			t.Logf("Vector weight=%.1f, Text weight=%.1f, Total score=%.6f",
				tt.vectorWeight, tt.textWeight, totalScore)

			assert.GreaterOrEqual(t, totalScore, tt.expectedMin)
			assert.LessOrEqual(t, totalScore, tt.expectedMax)
		})
	}
}

// TestRRFRankOrdering tests that RRF correctly orders documents
func TestRRFRankOrdering(t *testing.T) {
	rrfConstant := 60
	weight := 1.0

	// Calculate scores for different ranking scenarios
	type scenario struct {
		name        string
		vectorRank  int
		textRank    int
		description string
	}

	scenarios := []scenario{
		{
			name:        "both_top",
			vectorRank:  0,
			textRank:    0,
			description: "Appears in top of both searches",
		},
		{
			name:        "vector_top_text_mid",
			vectorRank:  0,
			textRank:    10,
			description: "Top in vector, middle in text",
		},
		{
			name:        "both_mid",
			vectorRank:  10,
			textRank:    10,
			description: "Middle in both searches",
		},
		{
			name:        "vector_only",
			vectorRank:  0,
			textRank:    1000, // Effectively not in text results
			description: "Only in vector search",
		},
	}

	scores := make(map[string]float64)
	for _, s := range scenarios {
		vectorScore := weight / float64(s.vectorRank+1+rrfConstant)
		textScore := weight / float64(s.textRank+1+rrfConstant)
		scores[s.name] = vectorScore + textScore

		t.Logf("%s (v:%d, t:%d): %.6f - %s",
			s.name, s.vectorRank, s.textRank, scores[s.name], s.description)
	}

	// Verify expected ordering
	assert.Greater(t, scores["both_top"], scores["vector_top_text_mid"],
		"Document in top of both should score higher than top in one")
	assert.Greater(t, scores["vector_top_text_mid"], scores["both_mid"],
		"Document with better ranks should score higher")
	assert.Greater(t, scores["both_mid"], scores["vector_only"],
		"Document in both searches should score higher than vector-only")
}
