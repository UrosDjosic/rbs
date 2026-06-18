## Projekat iz razvoja bezbednosti softvera - Oblak

Minimalni “platform skeleton” za serverless Python funkcije.

- **API** (Go): health, login, auth, upload funkcija, verifier, publish, invoke (stub)
- **CLI** (Go): `login`, `deploy`, `publish`, `list`, `invoke`
- **Verifier**: strukturni AV + ClamAV (opciono) + Bandit + requirements policy + pip-audit

### Preduslovi na serveru

**Bandit** (obavezno):

```powershell
pip install bandit
python -m bandit --version
```

**pip-audit** (obavezno ako `requirements.txt` ima zavisnosti):

```powershell
pip install pip-audit
pip-audit --version
```

**ClamAV** (opciono — ako nije dostupan, sloj se preskače):

Virus baze idu u `storage/clamav/database` (projekat, ne sistemski folder).

**Windows** — instaliraj **x64** (ne ARM64) sa [clamav.net/downloads](https://www.clamav.net/downloads):

```powershell
.\scripts\setup-clamav.ps1
```

**Linux** — instaliraj paket, pa skripta:

```bash
sudo apt install clamav clamav-freshclam   # Debian/Ubuntu
chmod +x scripts/setup-clamav.sh
./scripts/setup-clamav.sh
```

Provera: `clamscan --version` (baze: `storage/clamav/database`)

### Pokretanje

```powershell
go run ./cmd/api
```

Drugi terminal:

```powershell
go run ./cmd/cli login --user admin --pass admin
go run ./cmd/cli deploy --path .\samples\benign\hello_world --name hello_world
go run ./cmd/cli publish <function_id>
go run ./cmd/cli invoke <function_id>
```

### Verifier pipeline

Posle uploada API automatski pokreće:

1. **structural_av** — unpack + policy (ekstenzije, path traversal, magic bytes)
2. **clamav** — `clamscan -r workdir` (ako nije instaliran → preskočeno)
3. **static_bandit** — Bandit JSON sken (`LOW+` severity → reject)
4. **requirements_policy** — samo pinovane zavisnosti (`pkg==1.2.3`), bez git/URL/index
5. **dependency_audit** — `pip-audit -r requirements.txt` (CVE → reject; preskočeno ako nema deps)

Status verzije: `verified` ili `rejected`. `publish` dozvoljen samo za `verified`.

Ručno testiranje:

```powershell
go run ./cmd/verifier --zip storage\functions\<fn>\<ver>\src.zip
go run ./cmd/verifier --zip path\to\src.zip --skip-bandit
```

### Test primeri

| Sample | Rezultat |
|---|---|
| `samples/benign/hello_world` | verified |
| `samples/malicious/subprocess_shell` | rejected (Bandit) |
| `samples/malicious/eval_exec` | rejected (Bandit) |
| `samples/malicious/missing_main` | rejected (structural) |
| `samples/malicious/forbidden_script` | rejected (structural) |
| `samples/malicious/nested_main` | rejected (structural) |
| `samples/malicious/unpinned_requirements` | rejected (requirements_policy) |

### Endpoints

- `GET /health`
- `GET /ui`
- `POST /auth/login`
- `GET /me`
- `POST /functions` — upload + verify
- `POST /functions/{id}/deploy` — publish (samo verified)
- `POST /invoke/{function_id}`

### Podrazumevani korisnik

- user: `admin`
- pass: `admin`
