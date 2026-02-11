package rag

import (
	"encoding/binary"
	"math"
)

// serializeEmbedding converts float32 slice to bytes
func serializeEmbedding(embedding []float32) []byte {
	bytes := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}
	return bytes
}

// deserializeEmbedding converts bytes to float32 slice
func deserializeEmbedding(bytes []byte) []float32 {
	embedding := make([]float32, len(bytes)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(bytes[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(vec1, vec2 []float32) float32 {
	if len(vec1) == 0 || len(vec2) == 0 || len(vec1) != len(vec2) {
		return 0
	}

	var dotProduct float32
	var magnitudeVec1 float32
	var magnitudeVec2 float32

	for i := range vec1 {
		dotProduct += vec1[i] * vec2[i]
		magnitudeVec1 += vec1[i] * vec1[i]
		magnitudeVec2 += vec2[i] * vec2[i]
	}

	if magnitudeVec1 == 0 || magnitudeVec2 == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(magnitudeVec1))) * float32(math.Sqrt(float64(magnitudeVec2))))
}
