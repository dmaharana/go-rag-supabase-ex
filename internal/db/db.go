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
	bun.BaseModel `bun:"table:documents,alias:d"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Content       string    `bun:"content,notnull"`
	Embedding     []float32 `bun:"embedding,notnull,type:vector(768)"`
}

func NewDB(sqldb *sql.DB) *bun.DB {
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	return db
}

func ConnectDB(cfg *config.DbConfig) (*sql.DB, error) {
	supabaseURL := fmt.Sprintf("postgresql://%s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)
	dsn := supabaseURL + "?sslmode=disable"

	return sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), pgdriver.WithPassword(cfg.Password))), nil
}

func InitDB(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*Document)(nil)).IfNotExists().Exec(ctx)
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

// drop table documents

func DropDocuments(ctx context.Context, db *bun.DB) error {
	_, err := db.NewDropTable().Model((*Document)(nil)).IfExists().Exec(ctx)
	return err
}
