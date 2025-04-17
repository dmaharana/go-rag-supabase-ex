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
