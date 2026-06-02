## Oblak

Platforma za upload i pozivanje korisničkog Python koda (serverless, Firecracker u planu).

### Pokretanje API

```powershell
go run ./cmd/api
```

- Health: `http://127.0.0.1:8080/health`
- GUI: `http://127.0.0.1:8080/ui/`

Podrazumevani korisnik: `admin` / `admin`

### CLI — ceo tok

```powershell
# 1) Login
go run ./cmd/cli login --url http://127.0.0.1:8080 --user admin --pass admin

# 2) Upload funkcije (ZIP foldera)
go run ./cmd/cli deploy --path .\samples\benign\hello_world --name hello_world

# 3) Publish — dobijaš invoke_url
go run ./cmd/cli publish <function_id>

# 4) Lista funkcija (vidi invoke_url ako je publish-ovano)
go run ./cmd/cli list

# 5) Invoke (stub — samo upisuje run u bazu, ne izvršava Python)
go run ./cmd/cli invoke <function_id>
```

### cURL (invoke — bez tokena)

Posle `publish`, pozovi `invoke_url`:

```powershell
curl -X POST http://127.0.0.1:8080/invoke/<function_id>
```

Očekivani odgovor (stub):

```json
{
  "run_id": "...",
  "function_id": "...",
  "status": "done",
  "message": "stub: Python execution not implemented yet (run recorded only)"
}
```

### Endpoints

| Metoda | Putanja | Auth | Opis |
|--------|---------|------|------|
| GET | `/health` | ne | health check |
| POST | `/auth/login` | ne | token |
| GET | `/me` | Bearer | trenutni korisnik |
| POST | `/functions` | Bearer | upload ZIP |
| GET | `/functions` | Bearer | lista |
| POST | `/functions/{id}/deploy` | Bearer | publish → `invoke_url` |
| POST | `/invoke/{function_id}` | ne | stub invoke |

### Storage

- Baza: `storage/oblak.db`
- ZIP: `storage/functions/<function_id>/<version_id>/src.zip`
- Runovi: tabela `runs`

### Stubovi (još ne rade)

- `cmd/verifier` — analiza koda
- `cmd/runner` — Firecracker izvršavanje
