package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	nethttp "net/http"

	httpapi "oblak/internal/api/http"
	"oblak/internal/api/store/sqlite"
	"oblak/internal/common/ids"
	"oblak/internal/runner"
	"oblak/internal/runner/firecracker"
	"oblak/internal/runner/local"
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

	// Initialize the execution runner
	// Default to local runner for development
	// Can be switched to Firecracker if FIRECRACKER_* env vars are set
	runnerInstance := initRunner()
	defer runnerInstance.Close()

	api := httpapi.Server{DB: db, PublicBaseURL: publicURL, Runner: runnerInstance}
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

func initRunner() runner.Runner {
	// Check for Firecracker configuration
	fcKernel := os.Getenv("FIRECRACKER_KERNEL")
	fcRootfs := os.Getenv("FIRECRACKER_ROOTFS")

	if fcKernel != "" && fcRootfs != "" {
		// Try to use Firecracker runner
		log.Printf("Attempting to initialize Firecracker runner...")
		log.Printf("  Kernel: %s", fcKernel)
		log.Printf("  Rootfs: %s", fcRootfs)

		// Import firecracker runner only if needed
		// This avoids hard dependency on firecracker package
		fcRunner, err := initFirecrackerRunner(fcKernel, fcRootfs)
		if err != nil {
			log.Printf("Warning: Failed to initialize Firecracker runner: %v", err)
			log.Printf("Falling back to local runner")
			return local.NewLocalRunner("")
		}
		return fcRunner
	}

	// Default to local runner
	log.Printf("Using local runner (subprocess execution)")
	return local.NewLocalRunner("")
}

func initFirecrackerRunner(kernelPath, rootfsPath string) (runner.Runner, error) {
	// Verify kernel and rootfs exist
	if _, err := os.Stat(kernelPath); err != nil {
		return nil, fmt.Errorf("kernel not found: %w", err)
	}
	if _, err := os.Stat(rootfsPath); err != nil {
		return nil, fmt.Errorf("rootfs not found: %w", err)
	}

	// Firecracker uses Unix sockets, which are not supported on WSL's /mnt/c
	// DrvFS mount. Keep VM scratch state on a Linux filesystem by default.
	runsDir := env("OBLAK_RUNS_DIR", filepath.Join(os.TempDir(), "oblak-runs"))
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create runs directory: %w", err)
	}

	// Initialize Firecracker runner
	fcRunner, err := firecracker.NewFirecrackerRunner(kernelPath, rootfsPath, runsDir)
	if err != nil {
		return nil, fmt.Errorf("firecracker initialization failed: %w", err)
	}

	log.Printf("Firecracker runner initialized successfully")
	log.Printf("  Kernel: %s", kernelPath)
	log.Printf("  Rootfs: %s", rootfsPath)
	log.Printf("  Runs:   %s", runsDir)

	return fcRunner, nil
}
