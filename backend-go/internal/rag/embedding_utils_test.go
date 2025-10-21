package rag

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSerializeEmbedding_Basic tests basic serialization of float32 slices
func TestSerializeEmbedding_Basic(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
		wantLen   int
	}{
		{
			name:      "empty vector",
			embedding: []float32{},
			wantLen:   0,
		},
		{
			name:      "single value",
			embedding: []float32{1.0},
			wantLen:   4,
		},
		{
			name:      "multiple values",
			embedding: []float32{1.0, 2.0, 3.0},
			wantLen:   12,
		},
		{
			name:      "negative values",
			embedding: []float32{-1.5, -2.5, -3.5},
			wantLen:   12,
		},
		{
			name:      "zero values",
			embedding: []float32{0.0, 0.0, 0.0},
			wantLen:   12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := serializeEmbedding(tt.embedding)
			assert.Len(t, result, tt.wantLen)
			assert.Equal(t, len(tt.embedding)*4, len(result))
		})
	}
}

// TestSerializeEmbedding_SpecialValues tests serialization with special float values
func TestSerializeEmbedding_SpecialValues(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
	}{
		{
			name:      "positive infinity",
			embedding: []float32{float32(math.Inf(1))},
		},
		{
			name:      "negative infinity",
			embedding: []float32{float32(math.Inf(-1))},
		},
		{
			name:      "NaN",
			embedding: []float32{float32(math.NaN())},
		},
		{
			name:      "mixed special values",
			embedding: []float32{float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN())},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := serializeEmbedding(tt.embedding)
			assert.NotNil(t, result)
			assert.Equal(t, len(tt.embedding)*4, len(result))
		})
	}
}

// TestDeserializeEmbedding_Basic tests basic deserialization
func TestDeserializeEmbedding_Basic(t *testing.T) {
	tests := []struct {
		name     string
		bytes    []byte
		wantLen  int
		wantVals []float32
	}{
		{
			name:     "empty bytes",
			bytes:    []byte{},
			wantLen:  0,
			wantVals: []float32{},
		},
		{
			name:     "single float",
			bytes:    []byte{0x00, 0x00, 0x80, 0x3f}, // 1.0 in little-endian
			wantLen:  1,
			wantVals: []float32{1.0},
		},
		{
			name:     "multiple floats",
			bytes:    make([]byte, 12), // 3 floats
			wantLen:  3,
			wantVals: nil, // will check length only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deserializeEmbedding(tt.bytes)
			assert.Len(t, result, tt.wantLen)

			if tt.wantVals != nil {
				for i, v := range tt.wantVals {
					assert.InDelta(t, v, result[i], 0.0001)
				}
			}
		})
	}
}

// TestSerializeDeserialize_RoundTrip tests round-trip serialization
func TestSerializeDeserialize_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
	}{
		{
			name:      "empty",
			embedding: []float32{},
		},
		{
			name:      "single value",
			embedding: []float32{1.5},
		},
		{
			name:      "small vector",
			embedding: []float32{1.0, 2.0, 3.0, 4.0, 5.0},
		},
		{
			name:      "typical embedding size",
			embedding: make([]float32, 384), // Common embedding dimension
		},
		{
			name:      "large vector",
			embedding: make([]float32, 1536), // OpenAI embedding size
		},
		{
			name:      "negative values",
			embedding: []float32{-0.5, -1.5, -2.5},
		},
		{
			name:      "mixed values",
			embedding: []float32{-1.5, 0.0, 1.5, 3.14159, -2.71828},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize non-empty slices with values
			if len(tt.embedding) > 1 {
				for i := range tt.embedding {
					tt.embedding[i] = float32(i) * 0.1
				}
			}

			serialized := serializeEmbedding(tt.embedding)
			deserialized := deserializeEmbedding(serialized)

			require.Equal(t, len(tt.embedding), len(deserialized))
			for i := range tt.embedding {
				assert.Equal(t, tt.embedding[i], deserialized[i])
			}
		})
	}
}

// TestSerializeDeserialize_SpecialValues_RoundTrip tests special values round-trip
func TestSerializeDeserialize_SpecialValues_RoundTrip(t *testing.T) {
	embedding := []float32{
		float32(math.Inf(1)),
		float32(math.Inf(-1)),
		float32(math.NaN()),
	}

	serialized := serializeEmbedding(embedding)
	deserialized := deserializeEmbedding(serialized)

	require.Equal(t, len(embedding), len(deserialized))
	assert.True(t, math.IsInf(float64(deserialized[0]), 1))
	assert.True(t, math.IsInf(float64(deserialized[1]), -1))
	assert.True(t, math.IsNaN(float64(deserialized[2])))
}

// TestCosineSimilarity_IdenticalVectors tests cosine similarity with identical vectors
func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	tests := []struct {
		name   string
		vec    []float32
		wantSim float32
	}{
		{
			name:   "unit vector",
			vec:    []float32{1.0, 0.0, 0.0},
			wantSim: 1.0,
		},
		{
			name:   "random vector",
			vec:    []float32{1.5, 2.5, 3.5},
			wantSim: 1.0,
		},
		{
			name:   "negative values",
			vec:    []float32{-1.0, -2.0, -3.0},
			wantSim: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec, tt.vec)
			assert.InDelta(t, tt.wantSim, similarity, 0.0001)
		})
	}
}

// TestCosineSimilarity_OrthogonalVectors tests cosine similarity with orthogonal vectors
func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	tests := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{
			name: "2D orthogonal",
			vec1: []float32{1.0, 0.0},
			vec2: []float32{0.0, 1.0},
		},
		{
			name: "3D orthogonal",
			vec1: []float32{1.0, 0.0, 0.0},
			vec2: []float32{0.0, 1.0, 0.0},
		},
		{
			name: "scaled orthogonal",
			vec1: []float32{2.0, 0.0, 0.0},
			vec2: []float32{0.0, 3.0, 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, 0.0, similarity, 0.0001)
		})
	}
}

// TestCosineSimilarity_OppositeVectors tests cosine similarity with opposite vectors
func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	tests := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{
			name: "simple opposite",
			vec1: []float32{1.0, 2.0, 3.0},
			vec2: []float32{-1.0, -2.0, -3.0},
		},
		{
			name: "unit opposite",
			vec1: []float32{1.0, 0.0, 0.0},
			vec2: []float32{-1.0, 0.0, 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, -1.0, similarity, 0.0001)
		})
	}
}

// TestCosineSimilarity_KnownValues tests cosine similarity with known expected values
func TestCosineSimilarity_KnownValues(t *testing.T) {
	tests := []struct {
		name    string
		vec1    []float32
		vec2    []float32
		wantSim float32
	}{
		{
			name:    "45 degree angle in 2D",
			vec1:    []float32{1.0, 0.0},
			vec2:    []float32{1.0, 1.0},
			wantSim: float32(1.0 / math.Sqrt(2.0)), // cos(45°) ≈ 0.707
		},
		{
			name:    "30 degree angle approximation",
			vec1:    []float32{1.0, 0.0},
			vec2:    []float32{0.866, 0.5}, // cos(30°) ≈ 0.866
			wantSim: 0.866,
		},
		{
			name:    "similar direction vectors",
			vec1:    []float32{1.0, 2.0, 3.0},
			vec2:    []float32{2.0, 4.0, 6.0},
			wantSim: 1.0, // Parallel vectors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, tt.wantSim, similarity, 0.01)
		})
	}
}

// TestCosineSimilarity_EdgeCases tests edge cases for cosine similarity
func TestCosineSimilarity_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		vec1    []float32
		vec2    []float32
		wantSim float32
	}{
		{
			name:    "empty vectors",
			vec1:    []float32{},
			vec2:    []float32{},
			wantSim: 0.0,
		},
		{
			name:    "different lengths",
			vec1:    []float32{1.0, 2.0},
			vec2:    []float32{1.0, 2.0, 3.0},
			wantSim: 0.0,
		},
		{
			name:    "zero vector first",
			vec1:    []float32{0.0, 0.0, 0.0},
			vec2:    []float32{1.0, 2.0, 3.0},
			wantSim: 0.0,
		},
		{
			name:    "zero vector second",
			vec1:    []float32{1.0, 2.0, 3.0},
			vec2:    []float32{0.0, 0.0, 0.0},
			wantSim: 0.0,
		},
		{
			name:    "both zero vectors",
			vec1:    []float32{0.0, 0.0, 0.0},
			vec2:    []float32{0.0, 0.0, 0.0},
			wantSim: 0.0,
		},
		{
			name:    "single element vectors",
			vec1:    []float32{5.0},
			vec2:    []float32{3.0},
			wantSim: 1.0, // Same direction
		},
		{
			name:    "very small values",
			vec1:    []float32{0.0001, 0.0001, 0.0001},
			vec2:    []float32{0.0001, 0.0001, 0.0001},
			wantSim: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, tt.wantSim, similarity, 0.0001)
		})
	}
}

// TestCosineSimilarity_LargeVectors tests cosine similarity with realistic embedding sizes
func TestCosineSimilarity_LargeVectors(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{
			name: "384 dimensions (sentence-transformers)",
			size: 384,
		},
		{
			name: "768 dimensions (BERT)",
			size: 768,
		},
		{
			name: "1536 dimensions (OpenAI)",
			size: 1536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vec1 := make([]float32, tt.size)
			vec2 := make([]float32, tt.size)

			// Fill with sequential values
			for i := range vec1 {
				vec1[i] = float32(i) * 0.01
				vec2[i] = float32(i) * 0.01
			}

			similarity := cosineSimilarity(vec1, vec2)
			assert.InDelta(t, 1.0, similarity, 0.0001)
		})
	}
}

// TestCosineSimilarity_NormalizedVectors tests pre-normalized vectors
func TestCosineSimilarity_NormalizedVectors(t *testing.T) {
	// Unit vectors (already normalized)
	vec1 := []float32{0.6, 0.8, 0.0} // magnitude = 1
	vec2 := []float32{0.8, 0.6, 0.0} // magnitude = 1

	similarity := cosineSimilarity(vec1, vec2)

	// For unit vectors: cos(θ) = dot product
	expectedDot := vec1[0]*vec2[0] + vec1[1]*vec2[1] + vec1[2]*vec2[2]
	assert.InDelta(t, expectedDot, similarity, 0.0001)
}

// TestCosineSimilarity_Symmetry tests that similarity is symmetric
func TestCosineSimilarity_Symmetry(t *testing.T) {
	vec1 := []float32{1.5, 2.5, 3.5, 4.5}
	vec2 := []float32{2.0, 3.0, 4.0, 5.0}

	sim12 := cosineSimilarity(vec1, vec2)
	sim21 := cosineSimilarity(vec2, vec1)

	assert.Equal(t, sim12, sim21)
}

// TestCosineSimilarity_CommutativeProperty tests commutative property
func TestCosineSimilarity_CommutativeProperty(t *testing.T) {
	tests := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{
			name: "positive values",
			vec1: []float32{1.0, 2.0, 3.0},
			vec2: []float32{4.0, 5.0, 6.0},
		},
		{
			name: "mixed values",
			vec1: []float32{-1.0, 2.0, -3.0},
			vec2: []float32{4.0, -5.0, 6.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim1 := cosineSimilarity(tt.vec1, tt.vec2)
			sim2 := cosineSimilarity(tt.vec2, tt.vec1)
			assert.Equal(t, sim1, sim2)
		})
	}
}

// TestCosineSimilarity_BoundedRange tests that similarity is always in [-1, 1]
func TestCosineSimilarity_BoundedRange(t *testing.T) {
	tests := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{
			name: "random vectors 1",
			vec1: []float32{1.5, -2.3, 4.7, -0.8},
			vec2: []float32{-3.2, 1.1, -2.9, 5.4},
		},
		{
			name: "random vectors 2",
			vec1: []float32{100.0, -200.0, 300.0},
			vec2: []float32{-50.0, 75.0, -100.0},
		},
		{
			name: "very large values",
			vec1: []float32{1000.0, 2000.0, 3000.0},
			vec2: []float32{4000.0, 5000.0, 6000.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.GreaterOrEqual(t, similarity, float32(-1.0))
			assert.LessOrEqual(t, similarity, float32(1.0))
		})
	}
}

// TestSerializeEmbedding_LittleEndian tests that serialization uses little-endian
func TestSerializeEmbedding_LittleEndian(t *testing.T) {
	embedding := []float32{1.0}
	serialized := serializeEmbedding(embedding)

	// 1.0 in IEEE 754 float32 is 0x3f800000
	// In little-endian: [0x00, 0x00, 0x80, 0x3f]
	expected := []byte{0x00, 0x00, 0x80, 0x3f}
	assert.Equal(t, expected, serialized)
}

// TestDeserializeEmbedding_LittleEndian tests that deserialization uses little-endian
func TestDeserializeEmbedding_LittleEndian(t *testing.T) {
	// 1.0 in IEEE 754 float32 little-endian
	bytes := []byte{0x00, 0x00, 0x80, 0x3f}
	deserialized := deserializeEmbedding(bytes)

	require.Len(t, deserialized, 1)
	assert.Equal(t, float32(1.0), deserialized[0])
}

// TestSerializeEmbedding_DeterministicOutput tests that serialization is deterministic
func TestSerializeEmbedding_DeterministicOutput(t *testing.T) {
	embedding := []float32{1.5, 2.5, 3.5, 4.5, 5.5}

	result1 := serializeEmbedding(embedding)
	result2 := serializeEmbedding(embedding)

	assert.Equal(t, result1, result2)
}

// TestCosineSimilarity_NilVectors tests behavior with nil vectors (treated as empty)
func TestCosineSimilarity_NilVectors(t *testing.T) {
	var nilVec []float32
	normalVec := []float32{1.0, 2.0, 3.0}

	tests := []struct {
		name    string
		vec1    []float32
		vec2    []float32
		wantSim float32
	}{
		{
			name:    "first nil",
			vec1:    nilVec,
			vec2:    normalVec,
			wantSim: 0.0,
		},
		{
			name:    "second nil",
			vec1:    normalVec,
			vec2:    nilVec,
			wantSim: 0.0,
		},
		{
			name:    "both nil",
			vec1:    nilVec,
			vec2:    nilVec,
			wantSim: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := cosineSimilarity(tt.vec1, tt.vec2)
			assert.Equal(t, tt.wantSim, similarity)
		})
	}
}

// BenchmarkSerializeEmbedding benchmarks serialization performance
func BenchmarkSerializeEmbedding(b *testing.B) {
	sizes := []int{384, 768, 1536}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			embedding := make([]float32, size)
			for i := range embedding {
				embedding[i] = float32(i) * 0.01
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = serializeEmbedding(embedding)
			}
		})
	}
}

// BenchmarkDeserializeEmbedding benchmarks deserialization performance
func BenchmarkDeserializeEmbedding(b *testing.B) {
	sizes := []int{384, 768, 1536}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			embedding := make([]float32, size)
			for i := range embedding {
				embedding[i] = float32(i) * 0.01
			}
			serialized := serializeEmbedding(embedding)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = deserializeEmbedding(serialized)
			}
		})
	}
}

// BenchmarkCosineSimilarity benchmarks cosine similarity performance
func BenchmarkCosineSimilarity(b *testing.B) {
	sizes := []int{384, 768, 1536}

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			vec1 := make([]float32, size)
			vec2 := make([]float32, size)
			for i := range vec1 {
				vec1[i] = float32(i) * 0.01
				vec2[i] = float32(i) * 0.02
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cosineSimilarity(vec1, vec2)
			}
		})
	}
}

// BenchmarkSerializeDeserialize_RoundTrip benchmarks round-trip performance
func BenchmarkSerializeDeserialize_RoundTrip(b *testing.B) {
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.01
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialized := serializeEmbedding(embedding)
		_ = deserializeEmbedding(serialized)
	}
}
