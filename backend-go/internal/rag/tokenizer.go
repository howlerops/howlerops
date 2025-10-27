package rag

// Tokenizer is a tiny placeholder for sentence-transformer tokenization.
// In a full implementation this would be a WordPiece/BPE tokenizer.
type Tokenizer struct{}

func NewTokenizer() *Tokenizer { return &Tokenizer{} }

// Tokenize returns dummy token ids and attention mask sized to input length.
func (t *Tokenizer) Tokenize(text string) ([]int64, []int64) {
    n := len(text)
    if n == 0 {
        return []int64{101, 102}, []int64{1, 1} // [CLS][SEP]
    }
    if n > 128 {
        n = 128
    }
    ids := make([]int64, n)
    mask := make([]int64, n)
    for i := 0; i < n; i++ {
        ids[i] = int64(100 + (i % 50))
        mask[i] = 1
    }
    return ids, mask
}


