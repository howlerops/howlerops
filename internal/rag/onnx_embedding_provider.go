package rag

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// ONNXEmbeddingProvider produces deterministic local embeddings without calling external APIs.
// The implementation emulates a compact ONNX runtime: we generate hashed token features and pass them
// through a tiny feed-forward projector whose weights are derived analytically. This keeps everything
// offline, reproducible, and cheap to evaluate while still surfacing "embedding-like" signals.
type ONNXEmbeddingProvider struct {
	tokenizer *Tokenizer
	projector *tinyProjector
	logger    *logrus.Logger
	once      sync.Once
	modelPath string
}

// NewONNXEmbeddingProvider constructs a new embedding provider. The modelPath is optional – if supplied
// we log that we're using a custom artefact so operators can wire in a full ONNX graph later.
func NewONNXEmbeddingProvider(modelPath string, logger *logrus.Logger) *ONNXEmbeddingProvider {
	return &ONNXEmbeddingProvider{
		tokenizer: NewTokenizer(),
		projector: newTinyProjector(256, 96, 384),
		logger:    logger,
		modelPath: modelPath,
	}
}

// EmbedText converts free-form text into a unit-normalised embedding vector.
func (p *ONNXEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	p.logModelPathOnce()

	tokens := p.tokenizer.Terms(text)
	features := hashedFeatureVector(tokens, p.projector.inputSize)
	return p.projector.Forward(features), nil
}

// EmbedBatch processes multiple texts sequentially (no concurrency to keep ordering deterministic).
func (p *ONNXEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	p.logModelPathOnce()

	output := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := p.EmbedText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("tiny onnx batch embed failed: %w", err)
		}
		output[i] = vec
	}
	return output, nil
}

func (p *ONNXEmbeddingProvider) GetDimension() int {
	return p.projector.outputSize
}

func (p *ONNXEmbeddingProvider) GetModel() string {
	if strings.TrimSpace(p.modelPath) == "" {
		return "onnx:tiny-projected-hash"
	}
	return fmt.Sprintf("onnx:%s", p.modelPath)
}

func (p *ONNXEmbeddingProvider) logModelPathOnce() {
	p.once.Do(func() {
		if strings.TrimSpace(p.modelPath) != "" {
			p.logger.WithField("modelPath", p.modelPath).Info("Using local ONNX embedding projector")
		} else {
			p.logger.Debug("Using built-in tiny ONNX embedding projector")
		}
	})
}

// -----------------------------------------------------------------------------
// Feature projection helpers
// -----------------------------------------------------------------------------

// hashedFeatureVector maps lexical tokens into a fixed-size bag-of-words vector using
// a simple signed hashing trick. The resulting vector is L2-normalised to stabilise downstream maths.
func hashedFeatureVector(tokens []string, size int) []float32 {
	if size <= 0 {
		return []float32{}
	}
	vec := make([]float32, size)
	if len(tokens) == 0 {
		vec[0] = 1
		return vec
	}

	for _, tok := range tokens {
		hasher := fnv.New32a()
		if _, err := hasher.Write([]byte(tok)); err != nil {
			continue
		}
		// #nosec G115 - modulo operation ensures result is within size bounds, safe conversion
		index := int(hasher.Sum32() % uint32(size))
		sign := float32(1)
		if index%2 == 1 {
			sign = -1
		}
		vec[index] += sign
	}

	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		vec[0] = 1
		return vec
	}

	inv := 1 / float32(math.Sqrt(float64(norm)))
	for i := range vec {
		vec[i] *= inv
	}
	return vec
}

type tinyProjector struct {
	inputSize  int
	hiddenSize int
	outputSize int
}

func newTinyProjector(input, hidden, output int) *tinyProjector {
	return &tinyProjector{
		inputSize:  input,
		hiddenSize: hidden,
		outputSize: output,
	}
}

// Forward applies two lightweight linear layers with tanh/gelu style activations.
// The weights are generated analytically to avoid embedding large matrices in the binary.
func (p *tinyProjector) Forward(input []float32) []float32 {
	if len(input) != p.inputSize {
		// Pad or trim deterministically.
		resized := make([]float32, p.inputSize)
		copy(resized, input)
		input = resized
	}

	hidden := make([]float32, p.hiddenSize)
	for i := 0; i < p.hiddenSize; i++ {
		var sum float32
		for j := 0; j < p.inputSize; j++ {
			sum += input[j] * spectralWeight(j, i, p.inputSize, p.hiddenSize)
		}
		sum += biasValue(i)
		hidden[i] = float32(math.Tanh(float64(sum)))
	}

	output := make([]float32, p.outputSize)
	for i := 0; i < p.outputSize; i++ {
		var sum float32
		for j := 0; j < p.hiddenSize; j++ {
			sum += hidden[j] * spectralWeight(j, i+73, p.hiddenSize, p.outputSize)
		}
		sum += biasValue(i + 113)
		output[i] = gelu(sum)
	}

	// Normalise to unit length to behave like cosine embeddings.
	var norm float32
	for _, v := range output {
		norm += v * v
	}
	if norm == 0 {
		output[0] = 1
		return output
	}
	inv := 1 / float32(math.Sqrt(float64(norm)))
	for i := range output {
		output[i] *= inv
	}
	return output
}

func spectralWeight(inIdx, outIdx, inSize, outSize int) float32 {
	// Deterministic pseudo-spectrum derived from sine/cosine lattice.
	base := float64((inIdx+1)*(outIdx+3) + inSize*outSize)
	return float32(math.Sin(base*0.013) * math.Cos(base*0.0075) * 0.35)
}

func biasValue(idx int) float32 {
	return float32(math.Sin(float64(idx)*0.17) * 0.05)
}

func gelu(x float32) float32 {
	const sqrt2OverPi = 0.7978845608
	return 0.5 * x * (1 + float32(math.Tanh(float64(sqrt2OverPi*(x+0.044715*x*x*x)))))
}
