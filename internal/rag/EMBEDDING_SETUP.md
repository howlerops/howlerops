# Embedding Setup

## Local Embeddings (Recommended - Free)

### Install Ollama

**macOS:**
```bash
brew install ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
Download from [https://ollama.ai/download](https://ollama.ai/download)

### Start Ollama

```bash
ollama serve
```

The embedding model will auto-download on first use (~200MB for nomic-embed-text).

### Available Models

| Model | Size | Dimensions | Use Case |
|-------|------|------------|----------|
| nomic-embed-text | 137M | 768 | **Recommended** - Best balance of speed and quality |
| mxbai-embed-large | 335M | 1024 | Higher quality, slower inference |
| all-minilm | 23M | 384 | Fastest, smallest - good for testing |

### Configuration

In `config.yaml` (or via environment variables):

```yaml
rag:
  embedding:
    provider: "ollama"  # Use local Ollama
    ollama:
      endpoint: "http://localhost:11434"
      model: "nomic-embed-text"
      dimension: 768  # Must match model dimension
      auto_pull: true  # Auto-download model on first use
    cache:
      max_size: 10000  # Cache up to 10k embeddings
      ttl: "24h"       # Cache for 24 hours
```

**Environment Variables:**

```bash
export SQL_STUDIO_RAG_EMBEDDING_PROVIDER="ollama"
export SQL_STUDIO_RAG_EMBEDDING_OLLAMA_ENDPOINT="http://localhost:11434"
export SQL_STUDIO_RAG_EMBEDDING_OLLAMA_MODEL="nomic-embed-text"
export SQL_STUDIO_RAG_EMBEDDING_OLLAMA_DIMENSION="768"
export SQL_STUDIO_RAG_EMBEDDING_OLLAMA_AUTO_PULL="true"
```

## OpenAI Embeddings (Optional - Costs Money)

If you prefer OpenAI's embedding API:

```yaml
rag:
  embedding:
    provider: "openai"
    openai:
      model: "text-embedding-3-small"
      dimension: 1536
      api_key_env: "OPENAI_API_KEY"  # Environment variable name
    cache:
      max_size: 10000
      ttl: "24h"
```

Set your API key:
```bash
export OPENAI_API_KEY="sk-..."
```

**Cost:** OpenAI charges per token. The cache helps reduce costs by avoiding duplicate embeddings.

## Model Selection Guide

### For Development/Testing
- **all-minilm** (384 dims): Fastest, great for rapid development
- Low quality but good enough for testing

### For Production
- **nomic-embed-text** (768 dims): Best balance - recommended
- High quality, reasonable speed, free
- Works completely offline after download

### For Maximum Quality
- **mxbai-embed-large** (1024 dims): Highest quality
- Slower inference, larger model
- Use if you need best possible semantic search

### For API-First
- **OpenAI text-embedding-3-small**: Cloud-based
- No local setup needed
- Costs money per API call
- Best integration if already using OpenAI

## Usage Example

```go
package main

import (
    "context"

    "github.com/jbeck018/howlerops/backend-go/pkg/rag"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    ctx := context.Background()

    // Setup Ollama provider
    endpoint := "http://localhost:11434"
    model := "nomic-embed-text"
    dimension := 768

    // Ensure model is downloaded
    modelMgr := rag.NewOllamaModelManager(endpoint, logger)
    if err := modelMgr.EnsureModelAvailable(ctx, model); err != nil {
        logger.Fatal("Failed to ensure model:", err)
    }

    // Create provider and service
    provider := rag.NewOllamaEmbeddingProvider(endpoint, model, dimension, logger)
    service := rag.NewEmbeddingService(provider, logger)

    // Embed text
    text := "SELECT * FROM users WHERE status = 'active'"
    embedding, err := service.EmbedText(ctx, text)
    if err != nil {
        logger.Fatal("Failed to embed:", err)
    }

    logger.Infof("Generated %d-dimensional embedding", len(embedding))

    // Check cache stats
    stats := service.GetCacheStats()
    logger.Infof("Cache: %d hits, %d misses, %.2f%% hit rate",
        stats.Hits, stats.Misses, stats.HitRate*100)
}
```

## Performance Characteristics

### Ollama (nomic-embed-text)
- **First embedding:** 50-100ms (model load)
- **Subsequent embeddings:** 10-30ms
- **Batch of 10:** ~100-200ms
- **Memory:** ~500MB RAM

### Cache Performance
- **Cache hit:** <1ms (memory lookup)
- **Cache miss:** Full embedding time
- **Typical hit rate:** 70-90% in production

## Troubleshooting

### "Ollama not available" error

1. Check Ollama is running:
```bash
curl http://localhost:11434/api/tags
```

2. Start Ollama if not running:
```bash
ollama serve
```

### "Model not found" error

The model should auto-download if `auto_pull: true`. If not:

```bash
ollama pull nomic-embed-text
```

### "Dimension mismatch" error

Ensure `dimension` config matches model:
- nomic-embed-text: 768
- mxbai-embed-large: 1024
- all-minilm: 384

### Slow first request

This is normal - Ollama loads the model on first use. Subsequent requests are fast.

### High memory usage

Ollama keeps the model in memory. To reduce:
```bash
# Unload model when not in use
ollama stop nomic-embed-text
```

## Testing

Run the test suite:

```bash
cd backend-go

# Ensure Ollama is running
ollama serve

# Run integration tests
go test -v -race ./internal/rag/... -run TestOllama

# Run all RAG tests
go test -v -race ./internal/rag/...

# Benchmark embeddings
go test -bench=. ./internal/rag/... -run=^$
```

## Production Deployment

### Docker Compose

```yaml
services:
  ollama:
    image: ollama/ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama-models:/root/.ollama
    environment:
      - OLLAMA_HOST=0.0.0.0:11434

  backend:
    build: .
    environment:
      - SQL_STUDIO_RAG_EMBEDDING_PROVIDER=ollama
      - SQL_STUDIO_RAG_EMBEDDING_OLLAMA_ENDPOINT=http://ollama:11434
    depends_on:
      - ollama

volumes:
  ollama-models:
```

### Kubernetes

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ollama
spec:
  selector:
    app: ollama
  ports:
    - port: 11434
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: ollama
        image: ollama/ollama
        ports:
        - containerPort: 11434
        volumeMounts:
        - name: models
          mountPath: /root/.ollama
      volumes:
      - name: models
        persistentVolumeClaim:
          claimName: ollama-models
```

## Migration from Mock Implementation

The previous mock implementation returned sequential numbers:
```go
// OLD (WRONG)
for i := range embedding {
    embedding[i] = float32(i) / float32(dimension)
}
```

Ollama returns actual semantic embeddings based on text meaning. This enables:
- Semantic search (find similar queries/schemas)
- Context-aware RAG (retrieve relevant documentation)
- Intelligent query suggestions
- Schema understanding

No code changes needed - just configure Ollama and restart.
