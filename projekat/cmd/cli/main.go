package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"oblak/internal/cli/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "login":
		loginCmd(os.Args[2:])
	case "register":
		registerCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "deploy":
		deployCmd(os.Args[2:])
	case "list":
		listCmd(os.Args[2:])
	case "publish":
		publishCmd(os.Args[2:])
	case "invoke":
		invokeCmd(os.Args[2:])
	case "run":
		runCmd(os.Args[2:])
	case "help":
		helpCmd(os.Args[2:])
	case "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q (run: oblak help)\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(strings.TrimSpace(`
Oblak CLI — upravljanje funkcijama na serveru

Pokretanje (razvoj):
  go run ./cmd/cli <komanda> [opcije]

Pokretanje (posle build-a):
  go build -o oblak.exe ./cmd/cli
  .\oblak.exe <komanda> [opcije]

Tipičan redosled:
  1. register  → kreiraj nalog
  2. login     → sačuva token
  3. deploy    → upload ZIP (folder sa main.py)
  4. publish   → dobijaš invoke_url
  5. invoke    → poziv funkcije (stub)
  list         → pregled funkcija

Komande:
  help              prikaži ovu pomoć (ili: help <komanda>)
  register          registracija novog naloga
  login             prijava na API
  status            provera tokena (/me)
  deploy            upload funkcije (folder → zip)
  publish <id>      objavi funkciju → invoke_url
  list              lista uploadovanih funkcija
  invoke <id>       pozovi /invoke/<id> (opciono --data JSON)
  run <run_id>      pregled sačuvanog izvršavanja

Primeri:
  go run ./cmd/cli register --user john --pass mypass
  go run ./cmd/cli login --user john --pass mypass
  go run ./cmd/cli deploy --path .\samples\benign\hello_world --name hello_world
  go run ./cmd/cli publish AEqPCIngsHs-9TG1gfjplw
  go run ./cmd/cli invoke AEqPCIngsHs-9TG1gfjplw

Za detalje jedne komande:
  go run ./cmd/cli help deploy
`))
}

func helpCmd(args []string) {
	if len(args) == 0 {
		usage()
		return
	}
	switch args[0] {
	case "login":
		fmt.Println(strings.TrimSpace(`
login — prijava i čuvanje tokena lokalno

  go run ./cmd/cli login [--url http://127.0.0.1:8080] --user USER --pass PASS

Podrazumevano: admin / admin (kreira se pri prvom startu API-ja).
`))
	case "register":
		fmt.Println(strings.TrimSpace(`
register — registracija novog naloga

  go run ./cmd/cli register [--url http://127.0.0.1:8080] --user USER --pass PASS

Kreira novi nalog i automatski sačuva token lokalno.
Primeri:
  go run ./cmd/cli register --user john --pass mypass
  go run ./cmd/cli register --url http://api.example.com --user alice --pass secure123
`))
	case "status":
		fmt.Println("status — GET /me sa sačuvanim tokenom\n\n  go run ./cmd/cli status [--url http://127.0.0.1:8080]")
	case "deploy":
		fmt.Println(strings.TrimSpace(`
deploy — zipuje folder i šalje POST /functions

  go run ./cmd/cli deploy --path <folder> [--name ime] [--url ...]

Folder treba da sadrži main.py (i opciono requirements.txt).
`))
	case "publish":
		fmt.Println(strings.TrimSpace(`
publish — POST /functions/<id>/deploy → invoke_url

  go run ./cmd/cli publish <function_id> [--url ...]

function_id dobijaš iz output-a komande deploy ili iz list.
`))
	case "list":
		fmt.Println("list — GET /functions (zaštićeno tokenom)\n\n  go run ./cmd/cli list [--url ...]")
	case "invoke":
		fmt.Println(strings.TrimSpace(`
invoke — POST /invoke/<function_id> (javni endpoint)

  go run ./cmd/cli invoke <function_id> [--data '{"a":2,"b":2}'] [--url ...]

--data šalje JSON telo zahteva; funkcija ga čita sa stdin.
Koristi --data @file.json ako shell pojede navodnike.
Pre invoke mora publish (status deployed).
`))
	case "run":
		fmt.Println(strings.TrimSpace(`
run — GET /runs/<run_id> (zaštićeno tokenom)

  go run ./cmd/cli run <run_id> [--url ...]

Kratki pregled statusa, exit code-a i stdout/stderr iz baze.
run_id dobijaš iz output-a komande invoke.
`))
	default:
		fmt.Fprintf(os.Stderr, "nepoznata komanda za help: %q\n\n", args[0])
		usage()
		os.Exit(2)
	}
}

func loadConfig(urlOverride string) (config.Config, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, "", err
	}
	base := cfg.BaseURL
	if urlOverride != "" {
		base = urlOverride
	}
	if base == "" || cfg.Token == "" {
		return config.Config{}, "", fmt.Errorf("not logged in; run: oblak login ...")
	}
	return cfg, base, nil
}

func authedRequest(method, base, path string, body io.Reader, contentType string) ([]byte, int, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(method, strings.TrimRight(base, "/")+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func loginCmd(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	url := fs.String("url", "http://127.0.0.1:8080", "API base URL")
	user := fs.String("user", "", "username")
	pass := fs.String("pass", "", "password")
	_ = fs.Parse(args)

	if *user == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "user/pass required")
		os.Exit(2)
	}

	body, _ := json.Marshal(map[string]string{
		"username": *user,
		"password": *pass,
	})
	resp, err := http.Post(strings.TrimRight(*url, "/")+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "login failed: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &out); err != nil || out.Token == "" {
		fmt.Fprintln(os.Stderr, "bad response:", string(b))
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load failed:", err)
		os.Exit(1)
	}
	cfg.BaseURL = *url
	cfg.Token = out.Token
	if err := config.Save(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "config save failed:", err)
		os.Exit(1)
	}
	fmt.Println("ok: token saved")
}

func registerCmd(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	url := fs.String("url", "http://127.0.0.1:8080", "API base URL")
	user := fs.String("user", "", "username")
	pass := fs.String("pass", "", "password")
	_ = fs.Parse(args)

	if *user == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "user/pass required")
		os.Exit(2)
	}

	body, _ := json.Marshal(map[string]string{
		"username": *user,
		"password": *pass,
	})
	resp, err := http.Post(strings.TrimRight(*url, "/")+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "register failed: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &out); err != nil || out.Token == "" {
		fmt.Fprintln(os.Stderr, "bad response:", string(b))
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load failed:", err)
		os.Exit(1)
	}
	cfg.BaseURL = *url
	cfg.Token = out.Token
	if err := config.Save(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "config save failed:", err)
		os.Exit(1)
	}
	fmt.Println("ok: account created and token saved")
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	url := fs.String("url", "", "API base URL (optional; defaults to saved config)")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load failed:", err)
		os.Exit(1)
	}
	base := cfg.BaseURL
	if *url != "" {
		base = *url
	}
	if base == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not logged in; run: oblak login ...")
		os.Exit(1)
	}

	req, _ := http.NewRequest("GET", strings.TrimRight(base, "/")+"/me", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "status failed: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func deployCmd(args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	path := fs.String("path", "", "path to function folder (must contain main.py)")
	name := fs.String("name", "", "optional function name")
	url := fs.String("url", "", "API base URL (optional; defaults to saved config)")
	_ = fs.Parse(args)

	if *path == "" {
		fmt.Fprintln(os.Stderr, "--path is required")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load failed:", err)
		os.Exit(1)
	}
	base := cfg.BaseURL
	if *url != "" {
		base = *url
	}
	if base == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not logged in; run: oblak login ...")
		os.Exit(1)
	}

	zipBytes, err := zipDir(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "zip failed:", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if *name != "" {
		_ = mw.WriteField("name", *name)
	}
	fw, err := mw.CreateFormFile("zip", "src.zip")
	if err != nil {
		fmt.Fprintln(os.Stderr, "multipart failed:", err)
		os.Exit(1)
	}
	if _, err := fw.Write(zipBytes); err != nil {
		fmt.Fprintln(os.Stderr, "multipart write failed:", err)
		os.Exit(1)
	}
	_ = mw.Close()

	req, _ := http.NewRequest("POST", strings.TrimRight(base, "/")+"/functions", &buf)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "deploy failed: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func zipDir(dir string) ([]byte, error) {
	dir = filepath.Clean(dir)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// skip common junk and local invoke fixtures (not part of deployable function)
		if strings.HasPrefix(rel, ".git/") || strings.HasPrefix(rel, "storage/") {
			return nil
		}
		if rel == "payload.json" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = rel
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func listCmd(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	url := fs.String("url", "", "API base URL (optional)")
	_ = fs.Parse(args)

	_, base, err := loadConfig(*url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, code, err := authedRequest("GET", base, "/functions", nil, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	if code != 200 {
		fmt.Fprintf(os.Stderr, "list failed: HTTP %d: %s\n", code, string(b))
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func publishCmd(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	url := fs.String("url", "", "API base URL (optional)")
	_ = fs.Parse(args)

	if len(fs.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: oblak publish <function_id>")
		os.Exit(2)
	}
	fnID := fs.Args()[0]

	_, base, err := loadConfig(*url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, code, err := authedRequest("POST", base, "/functions/"+fnID+"/deploy", nil, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	if code != 200 {
		fmt.Fprintf(os.Stderr, "publish failed: HTTP %d: %s\n", code, string(b))
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func invokeCmd(args []string) {
	fnID, baseURL, payload, err := parseInvokeArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	base := "http://127.0.0.1:8080"
	if baseURL != "" {
		base = baseURL
	} else {
		cfg, err := config.Load()
		if err == nil && cfg.BaseURL != "" {
			base = cfg.BaseURL
		}
	}

	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	}
	resp, err := http.Post(strings.TrimRight(base, "/")+"/invoke/"+fnID, "application/json", body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "invoke failed: HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func parseInvokeArgs(args []string) (fnID, baseURL, payload string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--url":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("usage: oblak invoke <function_id> [--data JSON] [--url ...]")
			}
			baseURL = args[i+1]
			i++
		case "--data":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("usage: oblak invoke <function_id> [--data JSON] [--url ...]")
			}
			payload = args[i+1]
			if strings.HasPrefix(payload, "@") {
				b, readErr := os.ReadFile(strings.TrimPrefix(payload, "@"))
				if readErr != nil {
					return "", "", "", fmt.Errorf("read data file: %w", readErr)
				}
				payload = string(b)
			}
			i++
		case "-h", "--help":
			return "", "", "", fmt.Errorf("usage: oblak invoke <function_id> [--data JSON] [--url ...]")
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", fmt.Errorf("unknown flag: %s", args[i])
			}
			if fnID != "" {
				return "", "", "", fmt.Errorf("usage: oblak invoke <function_id> [--data JSON] [--url ...]")
			}
			fnID = args[i]
		}
	}
	if fnID == "" {
		return "", "", "", fmt.Errorf("usage: oblak invoke <function_id> [--data JSON] [--url ...]")
	}
	return fnID, baseURL, payload, nil
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	url := fs.String("url", "", "API base URL (optional)")
	_ = fs.Parse(args)

	if len(fs.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "usage: oblak run <run_id>")
		os.Exit(2)
	}
	runID := fs.Args()[0]

	_, base, err := loadConfig(*url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b, code, err := authedRequest("GET", base, "/runs/"+runID, nil, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	if code != 200 {
		fmt.Fprintf(os.Stderr, "run lookup failed: HTTP %d: %s\n", code, string(b))
		os.Exit(1)
	}

	var out struct {
		RunID           string  `json:"run_id"`
		FunctionID      string  `json:"function_id"`
		VersionID       string  `json:"version_id"`
		Status          string  `json:"status"`
		ExitCode        *int    `json:"exit_code"`
		Stdout          string  `json:"stdout"`
		Stderr          string  `json:"stderr"`
		StdoutTruncated bool    `json:"stdout_truncated"`
		StderrTruncated bool    `json:"stderr_truncated"`
		Message         *string `json:"message"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		fmt.Fprintln(os.Stderr, "bad response:", string(b))
		os.Exit(1)
	}

	exit := "?"
	if out.ExitCode != nil {
		exit = fmt.Sprintf("%d", *out.ExitCode)
	}
	fmt.Printf("run_id:     %s\n", out.RunID)
	fmt.Printf("function:   %s\n", out.FunctionID)
	fmt.Printf("version:    %s\n", out.VersionID)
	fmt.Printf("status:     %s (exit %s)\n", out.Status, exit)
	if out.Message != nil && *out.Message != "" {
		fmt.Printf("message:    %s\n", *out.Message)
	}
	fmt.Printf("stdout:     %s\n", displayStream(out.Stdout, out.StdoutTruncated))
	fmt.Printf("stderr:     %s\n", displayStream(out.Stderr, out.StderrTruncated))
}

func displayStream(s string, truncated bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(prazno)"
	}
	if truncated {
		return s + " [skraćeno]"
	}
	return s
}
