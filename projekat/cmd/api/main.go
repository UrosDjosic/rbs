package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	nethttp "net/http"

	httpapi "oblak/internal/api/http"
	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/ids"
)

func main() {
	addr := env("OBLAK_ADDR", "127.0.0.1:8080")
	dbPath := env("OBLAK_DB", filepath.Join("storage", "oblak.db"))

	if err := os.MkdirAll("storage", 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schemaSQL, err := os.ReadFile(filepath.Join("internal", "api", "store", "sqlite", "schema.sql"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if err := db.ExecSchema(ctx, string(schemaSQL)); err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	if err := ensureDefaultAdmin(db); err != nil {
		log.Fatal(err)
	}

	mux := nethttp.NewServeMux()
	ui := nethttp.FileServer(nethttp.Dir(filepath.Join("web", "static")))
	publicURL := env("OBLAK_PUBLIC_URL", "http://"+addr)
	api := httpapi.Server{DB: db, PublicBaseURL: publicURL}
	api.Register(mux, ui)

	srv := &nethttp.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("api listening on http://%s", addr)
	log.Printf("ui at http://%s/ui", addr)
	log.Fatal(srv.ListenAndServe())
}

func ensureDefaultAdmin(db *sqlite.DB) error {
	ctx := context.Background()
	existing, err := db.GetUserByUsername(ctx, "admin")
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	uid, err := ids.NewToken(16)
	if err != nil {
		return err
	}
	return db.InsertUser(ctx, sqlite.User{
		ID:           uid,
		Username:     "admin",
		PasswordHash: pwHash,
		CreatedAt:    time.Now(),
	})
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
