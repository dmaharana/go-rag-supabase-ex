package chromemdb

import (
	"context"
	"fmt"
	"runtime"

	"github.com/philippgille/chromem-go"
	"github.com/rs/zerolog/log"
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
	dbPath     string
	compress   bool
	encryptionKey string
	filePath   string
}

const (
	compress = false
)

// NewVectorDBManager initializes a new vector database manager
func NewVectorDBManager(dbPath,collectionName string, inMemory bool, encryptionKey string) (*VectorDBManager, error) {
	ctx := context.Background()
	var db *chromem.DB
	var err error
	if inMemory {
		db = chromem.NewDB()
	} else {
		db, err = chromem.NewPersistentDB(dbPath, compress)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %v", err)
		}
	}

	return &VectorDBManager{
		db:         db,
		collection: nil,
		ctx:        ctx,
		dbPath:     dbPath,
		compress:   compress,
		encryptionKey: encryptionKey,
		filePath:   dbPath + "/" + collectionName + ".chromem",
	}, nil
}

// create or read collection
func (m *VectorDBManager) GetOrCreateCollection(collectionName string) (*chromem.Collection, error) {
	c, 	err := m.db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create/get collection: %v", err)
	}
	m.collection = c
	return c, nil
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
func (m *VectorDBManager) SearchWithQueryOptions(ctx context.Context, opts chromem.QueryOptions) ([]chromem.Result, error) {
	// exit if query or embedding is not provided
	if opts.QueryText == "" && opts.QueryEmbedding == nil {
		return nil, fmt.Errorf("either query or embedding must be provided")
	}

	// Perform similarity search
	results, err := m.collection.QueryWithOptions(m.ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query by similarity: %v", err)
	}
	return results, nil
}

// delete collection
func (m *VectorDBManager) DeleteCollection() error {
	err := m.db.DeleteCollection(m.collection.Name)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %v", err)
	}
	return nil
}

// export to file
func (m *VectorDBManager) Export(ctx context.Context) error {
	if m.encryptionKey == "" {
		return fmt.Errorf("encryption key is required")
	}
	if m.collection == nil {
		return fmt.Errorf("collection is required")
	}
	if m.dbPath == "" {
		return fmt.Errorf("db path is required")
	}

	log.Debug().Msgf("Collection name: %s", m.collection.Name)
	log.Debug().Msgf("File path: %s", m.filePath)
	log.Debug().Msgf("Compress: %t", m.compress)
	log.Debug().Msgf("DB path: %s", m.dbPath)
	// export collection
	err := m.db.ExportToFile(m.filePath, m.compress, m.encryptionKey, m.collection.Name)
	if err != nil {
		return fmt.Errorf("failed to export database: %v", err)
	}
	return nil
}

// import from file
func (m *VectorDBManager) Import(ctx context.Context) error {
	err := m.db.ImportFromFile(m.dbPath, m.collection.Name)
	if err != nil {
		return fmt.Errorf("failed to import database: %v", err)
	}
	return nil
}
