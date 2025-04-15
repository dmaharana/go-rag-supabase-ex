package db

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type Document struct {
	bun.BaseModel `bun:"table:documents,alias:d"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Content       string    `bun:"content,notnull"`
	Embedding     []float32 `bun:"embedding,notnull,type:vector(1536)"`
}

func NewDB(supabaseURL, supabaseKey string) (*bun.DB, error) {
	dsn := supabaseURL + "?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), pgdriver.WithPassword(supabaseKey)))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	_, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS vector;
		CREATE TABLE IF NOT EXISTS documents (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			embedding VECTOR(1536) NOT NULL
		);
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func StoreDocument(ctx context.Context, db *bun.DB, content string, embedding []float32) error {
	doc := &Document{
		Content:   content,
		Embedding: embedding,
	}
	_, err := db.NewInsert().Model(doc).Exec(ctx)
	return err
}

func SearchDocuments(ctx context.Context, db *bun.DB, queryEmbedding []float32, limit int) ([]Document, error) {
	var docs []Document
	err := db.NewSelect().
		Model(&docs).
		Column("id", "content", "embedding").
		OrderExpr("embedding <-> ?", queryEmbedding).
		Limit(limit).
		Scan(ctx)
	return docs, err
}
