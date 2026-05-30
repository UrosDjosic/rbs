# Exploit chain — uputstvo

Timski POC za TUDO (login bypass → APE → RCE).

## Pre nego što pokreneš skriptu

Potreban ti je:
- Docker i Docker Compose
- Python 3
- `requests` (`pip3 install requests`)

Aplikacija mora da radi u kontejneru pre exploit-a.

## 1. Pokreni izazov (Docker)

Iz root foldera `tudo/`:

```bash
cd tudo
docker compose up -d
```

Proveri da li radi:

```bash
docker compose ps
```

Treba da vidiš `tudo-app`, `tudo-db` i `tudo-admin` kao **Up**.

Otvori u browseru: [http://localhost:8000](http://localhost:8000)

## 2. Pokreni exploit skriptu

Još uvek iz `tudo/` foldera:

```bash
python3 scripts/exploit_chain.py http://localhost:8000 user1 'nova_sifra'
```

Argumenti:
- URL aplikacije
- korisnik (`user1` ili `user2`, ne `admin`)
- nova lozinka koju postavljaš

Ako ne uspe iz prvog puta, probaj sa većim vremenskim prozorom:

```bash
python3 scripts/exploit_chain.py http://localhost:8000 user1 'nova_sifra' -p 100
```

## 3. Kako znaš da je uspelo

U terminalu:

```
[+] token: ...
[+] password set to nova_sifra
[+] logged in as user1
[+] step 1 done
```

U browseru: login na [http://localhost:8000/login.php](http://localhost:8000/login.php) sa novom lozinkom.

## Zaustavi / restartuj izazov

Zaustavi:

```bash
docker compose down
```

Ponovo pokreni (npr. posle greške ili da resetuješ bazu):

```bash
docker compose down
docker compose up -d
```

## Fajlovi

| Fajl | Šta radi |
|------|----------|
| `exploit_chain.py` | glavna skripta — Step 1 (auth), Step 2/3 TODO |
| `auth.py` | login bypass logika (AUTH-2) |
| `generateTokens.php` | generiše kandidate tokena (PHP helper) |

## Napomena

PHP na hostu nije obavezan — ako nemaš lokalni `php`, skripta koristi PHP iz `tudo-app` kontejnera automatski.
