package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loreline/loreline/services/gateway/internal/controlplane"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: lorelinectl migrate | bootstrap <tenant-slug> <tenant-name>")
	}
	db := os.Getenv("DATABASE_URL")
	if db == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	switch os.Args[1] {
	case "migrate":
		err = controlplane.Migrate(ctx, pool)
	case "bootstrap":
		if len(os.Args) < 4 {
			log.Fatal("bootstrap requires tenant slug and name")
		}
		err = bootstrap(ctx, pool, os.Args[2], strings.Join(os.Args[3:], " "))
	default:
		log.Fatal("unknown command")
	}
	if err != nil {
		log.Fatal(err)
	}
}
func bootstrap(ctx context.Context, pool *pgxpool.Pool, slug, name string) error {
	if err := controlplane.Migrate(ctx, pool); err != nil {
		return err
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return err
	}
	secret := "ll_live_" + base64.RawURLEncoding.EncodeToString(secretBytes)
	hash := sha256.Sum256([]byte(secret))
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var tenant string
	if err = tx.QueryRow(ctx, `INSERT INTO tenants(slug,name) VALUES($1,$2) ON CONFLICT(slug) DO UPDATE SET name=excluded.name RETURNING id::text`, slug, name).Scan(&tenant); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO api_keys(tenant_id,name,key_prefix,key_hash,role) VALUES($1,'bootstrap',$2,$3,'owner')`, tenant, secret[:16], hash[:]); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("tenant_id=%s\napi_key=%s\nStore this key now; it will not be shown again.\n", tenant, secret)
	return nil
}
