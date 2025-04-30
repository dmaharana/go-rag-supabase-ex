package models

// Chunk represents a parsed chunk with metadata
type Chunk struct {
	Content    string
	PageNumber int
	ChunkID    int
}

type PromptResponse struct {
	Query   string
	Source  string
	Content string
}

// ChunkEmbedding holds the content, embedding, and metadata for a single chunk
type ChunkEmbedding struct {
	Content        string
	Embedding      []float32
	SourceFilename string
	PageNumber     int // Nullable for non-paged formats
	ChunkID        int
}
