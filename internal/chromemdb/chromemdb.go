package chromemdb

import (
	"context"
	"document-rag/internal/helper"
	"document-rag/internal/models"
	"fmt"
	"runtime"

	"github.com/philippgille/chromem-go"
)

// Document represents our data structure with content and metadata
type Document struct {
	ID        string
	Content   string
	Metadata  map[string]string
	Embedding []float32
}

// meta data will have source filename, page number, chunk id

// VectorDBManager encapsulates the chromem-go database operations
type VectorDBManager struct {
	db         *chromem.DB
	collection *chromem.Collection
	ctx        context.Context
}

// NewVectorDBManager initializes a new vector database manager
func NewVectorDBManager(dbPath,collectionName string) (*VectorDBManager, error) {
	ctx := context.Background()
	compress := true
	// db := chromem.NewDB()
	db, err := chromem.NewPersistentDB(dbPath, compress)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}

	// Create or get collection with default embedding function
	collection, err := db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create/get collection: %v", err)
	}

	return &VectorDBManager{
		db:         db,
		collection: collection,
		ctx:        ctx,
	}, nil
}

// Create adds a new document with content, metadata, and optional embedding
func (m *VectorDBManager) Create(doc Document) error {
	chromemDoc := chromem.Document{
		ID:        doc.ID,
		Content:   doc.Content,
		Metadata:  doc.Metadata,
		Embedding: doc.Embedding,
	}

	err := m.collection.AddDocuments(m.ctx, []chromem.Document{chromemDoc}, runtime.NumCPU())
	if err != nil {
		return fmt.Errorf("failed to add document: %v", err)
	}
	return nil
}

// add multiple documents
func (m *VectorDBManager) CreateDocs(documents []chromem.Document) error {
	err := m.collection.AddDocuments(m.ctx, documents, runtime.NumCPU())
	if err != nil {
		return fmt.Errorf("failed to add document: %v", err)
	}
	return nil
}

// Read retrieves documents by ID or performs a similarity search
func (m *VectorDBManager) Read(id string, query string, limit int) ([]chromem.Result, error) {
	if id != "" {
		// Query by ID (exact match)
		results, err := m.collection.Query(m.ctx, "", 1, map[string]string{"id": id}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query by ID: %v", err)
		}
		return results, nil
	}

	if query != "" {
		// Perform similarity search
		results, err := m.collection.Query(m.ctx, query, limit, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query by similarity: %v", err)
		}
		return results, nil
	}

	return nil, fmt.Errorf("either id or query must be provided")
}

// delete collection
func (m *VectorDBManager) DeleteCollection() error {
	err := m.db.DeleteCollection(m.collection.Name)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %v", err)
	}
	return nil
}

// convert chunkEmbedding to chromem.Document
func (m *VectorDBManager) convertToDocument(chunkEmbeddings []models.ChunkEmbedding) []chromem.Document {
	var documents []chromem.Document
	for _, chunkEmbedding := range chunkEmbeddings {
		id, err := helper.GenerateUUID()
		if err != nil {
			id = fmt.Sprintf("chunk-%d", chunkEmbedding.ChunkID)
		}
		documents = append(documents, chromem.Document{
			ID:      id,
			Content: chunkEmbedding.Content,
			Metadata: map[string]string{
				"source_filename": chunkEmbedding.SourceFilename,
				"page_number":     fmt.Sprintf("%d", chunkEmbedding.PageNumber),
				"chunk_id":        fmt.Sprintf("%d", chunkEmbedding.ChunkID),
			},
			Embedding: chunkEmbedding.Embedding,
		})
	}
	return documents
}
