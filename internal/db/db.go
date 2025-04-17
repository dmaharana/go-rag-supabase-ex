package db

import (
	"context"
	"database/sql"
	"fmt"

	"document-rag/internal/config"

	_ "github.com/lib/pq"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type Document struct {
	bun.BaseModel  `bun:"table:documents,alias:d"`
	ID             int64     `bun:"id,pk,autoincrement"`
	Content        string    `bun:"content,notnull"`
	Embedding      []float32 `bun:"embedding,notnull"`
	SourceFilename string    `bun:"source_filename,notnull"`
	PageNumber     int       `bun:"page_number"` // Nullable for non-paged formats
	ChunkID        int       `bun:"chunk_id,notnull"`
}

func NewDB(sqldb *sql.DB, isVerbose bool) *bun.DB {
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(isVerbose)))

	return db
}

func ConnectDB(cfg *config.DbConfig) (*sql.DB, error) {
	supabaseURL := fmt.Sprintf("postgresql://%s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)
	dsn := supabaseURL + "?sslmode=disable"

	return sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), pgdriver.WithPassword(cfg.Password))), nil
}

func InitDB(ctx context.Context, db *bun.DB, vectorSize int) error {
	// Enable vector extension
	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to enable vector extension: %w", err)
	}

	// Create documents table using Bun's schema definition
	_, err = db.NewCreateTable().
		Model((*Document)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	// Check if embedding column needs to be updated
	var columnType string
	err = db.QueryRow(`
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'documents' AND column_name = 'embedding'
	`).Scan(&columnType)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check embedding column type: %w", err)
	}

	// If column exists but isn't VECTOR, drop and recreate it
	if columnType != "" && columnType != "vector" {
		_, err = db.Exec(`
		ALTER TABLE documents 
		DROP COLUMN embedding,
		ADD COLUMN embedding VECTOR(?)`,
			vectorSize,
		)
		if err != nil {
			return fmt.Errorf("failed to update embedding column to VECTOR(%d): %w", vectorSize, err)
		}
	} else if columnType == "" {
		// If column doesn't exist, add it
		_, err = db.Exec(
			`ALTER TABLE documents ADD COLUMN embedding VECTOR(?)`,
			vectorSize,
		)
		if err != nil {
			return fmt.Errorf("failed to add embedding column: %w", err)
		}
	}

	return err
}

func StoreDocument(ctx context.Context, db *bun.DB, content string, embedding []float32) error {
	doc := &Document{
		Content:   content,
		Embedding: embedding,
	}
	_, err := db.NewInsert().Model(doc).Exec(ctx)
	return err
}

func StoreDocuments(ctx context.Context, db *bun.DB, documents []Document) error {
	_, err := db.NewInsert().Model(&documents).Exec(ctx)
	return err
}

func SearchDocuments(ctx context.Context, db *bun.DB, queryEmbedding []float32, limit int) ([]Document, error) {
	var docs []Document
	err := db.NewSelect().
		Model(&docs).
		Column("id", "content", "source_filename", "page_number", "chunk_id").
		// OrderExpr("embedding <=> ?", queryEmbedding).
		OrderExpr("embedding <-> ?", queryEmbedding).
		Limit(limit).
		Scan(ctx)
	return docs, err
}

// drop table documents

func DropDocuments(ctx context.Context, db *bun.DB) error {
	_, err := db.NewDropTable().Model((*Document)(nil)).IfExists().Exec(ctx)
	return err
}
