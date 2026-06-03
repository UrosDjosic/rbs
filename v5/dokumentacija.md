**LOGIN BYPASS - Viktor Srbljin SV63/2022**



**Metod pregleda koda**



- Za statičku analizu nisam koristio alat, već ručno čitanje izvornog koda u VS Code-u.

- Na Linux-u sam mogao da koristim `grep` (npr. `grep -rn "generateToken" app/`), ali sam umesto toga otvorio fajlove jedan po jedan i pratio tok podataka od forme do baze.

- **Zašto:** PDF zadatak traži code review pristup; ručno čitanje omogućava da se vidi ceo flow, a ne samo pojedinačni match.



**Mapiranje auth površine**



- Na login stranici postoje linkovi: `login.php`, `forgotusername.php`, `forgotpassword.php`.

- Testirao sam **Forgot password** za `user1` - aplikacija prikazuje poruku "Email sent!", ali token se nigde ne prikazuje u browseru.

- **Zašto:** ako token ne stiže korisniku kroz UI, mora postojati drugi način da se dođe do njega (čitanje iz baze, SQLi, ili predviđanje generisanja).



**Flow zahteva za promenu šifre - od forme do baze**



1. **`forgotpassword.php` (POST)** - korisnik unosi `username`.

   - Provera da korisnik postoji u tabeli `users` (prepared statement).

   - Admin (`admin`) je eksplicitno blokiran.

   - Poziva se `generateToken()` iz `app/includes/utils.php`.

   - Token se upisuje u tabelu `tokens` (`uid`, `token`).

   - Korisnik vidi samo "Email sent!" - email se u Docker okruženju ne šalje.

2. **`resetpassword.php?token=...` (GET)** - provera da li token postoji u bazi.

   - Ako postoji -> prikaz forme za novu lozinku.

   - Ako ne -> poruka "Token is invalid."

3. **`resetpassword.php` (POST)** - korisnik šalje `token`, `password1`, `password2`.

   - Provera tokena u tabeli `tokens`.

   - `UPDATE users SET password = ... WHERE uid = ...` (SHA256 hash).

   - `DELETE FROM tokens WHERE token = ...` (token je jednokratan).

4. **`login.php` (POST)** - login sa novom lozinkom.



**Kako sam shvatio da se token čuva u bazi**



- U `docker/init.sql` postoji tabela `tokens` sa kolonama `uid` i `token`.

- U `forgotpassword.php` (linija 22-23) vidi se `INSERT INTO tokens`.

- Ručna provera u bazi potvrđuje:



```bash
docker exec -it tudo-db psql -U postgres tudo -c "SELECT * FROM tokens;"
```



- Posle *Forgot password* requesta u tabeli se pojavi novi red; posle uspešnog reset-a token nestane.

- **Zašto:** token je jedini ključ za promenu tuđe lozinke - ko ga ima, menja lozinku.



**Ranjivost - predvidiv token**



- Funkcija `generateToken()` u `app/includes/utils.php`:



```php
srand(round(microtime(true) * 1000));
// zatim 32x rand() → string od 32 karaktera
```



- Seed za `rand()` je trenutno vreme u milisekundama - nije kriptografski siguran generator.

- Ako znamo približno vreme kad je server generisao token, možemo lokalno generisati iste kandidate.

- **Zašto:** `srand()` + `rand()` nisu namenjeni za security tokene; seed je trivijalan za pogoditi u uskom vremenskom prozoru.



**Exploit skripta**



- Lokacija: `tudo/scripts/exploit_chain.py` (Step 1), logika u `tudo/scripts/auth.py`.

- Pokretanje:



```bash
cd tudo
python3 scripts/exploit_chain.py http://localhost:8000 user1 'nova_sifra'
```



- Koraci koje skripta radi:

  1. `POST /forgotpassword.php` - zabeleži vreme pre i posle requesta.

  2. `generateTokens.php` - generiše listu mogućih tokena za taj vremenski opseg (isti PHP algoritam kao aplikacija).

  3. `GET /resetpassword.php?token=...` - traži validan token.

  4. `POST /resetpassword.php` - postavlja novu lozinku.

  5. `POST /login.php` - potvrđuje da login radi.



**Rezime BYPASS provere**



- Tip ranjivosti: predvidiv password reset token (slaba randomizacija).

- Impact: napadač bez lozinke može resetovati nalog bilo kog korisnika osim `admin`.

- PoC: `exploit_chain.py` Step 1.



**Kako sprečiti ovu ranjivost**



- Koristiti kriptografski siguran generator, npr. PHP `random_bytes()` / `bin2hex(random_bytes(32))` ili `random_int()`.

- **Ne koristiti** `srand()` i `rand()` za security tokene.

- Token slati korisniku samo preko emaila (ili drugog kanala van aplikacije), sa rokom važenja.

- Ograničiti broj pokušaja reset-a po IP/korisniku (rate limiting).

- **Zašto:** `random_bytes()` daje entropiju koju napadač ne može predvideti iz vremena requesta.

** **

**ADMIN PRIVELEGE ESCALATION - Uros Djosic SV20/2022**



**Metod pregleda koda**



- Statička analiza ručnim čitanjem u VS Code-u, praćenjem toka podataka od HTTP parametra do SQL upita.
- Koristio sam `grep -rn "pg_query\b" app/` da nađem sve pozive koji ne koriste parameterizovane upite — svaki takav poziv je potencijalni SQLi kandidat.
- **Zašto:** `pg_query()` (za razliku od `pg_query_params()`) prima sirov SQL string i **podržava stacked queries** (više naredbi razdvojenih sa `;`), što znači da jedna injekcija može pokrenuti i `SELECT` i `UPDATE`/`DELETE` u istom zahtevu.



**Mapiranje admin površine**



- Na `/admin.php` postoji panel dostupan samo adminu — aplikacija vrši redirect na login ako `is_admin` nije `true` u bazi.
- Tabela `users` u `docker/init.sql` ima kolonu `is_admin BOOLEAN DEFAULT false`.
- Admin (`admin`) ima `is_admin = true`; regularni korisnici (`user1`, `user2`) imaju `false`.
- **Zašto:** Cilj je promeniti `is_admin` vrednost za naš nalog direktno u bazi, bez znanja admin lozinke.



**Pronalaženje ranjivog mesta — class.php**



- U `app/class.php` (linija ~18) nalazi se sledeći kod:

```php
$sql = "SELECT * FROM class_posts WHERE title = '" . $_POST['title'] . "'";
$result = pg_query($conn, $sql);
```

- Parametar `$_POST['title']` se direktno spaja u SQL string — **nema prepared statement-a, nema escapovanja**.
- Isti fajl koristi `pg_query()` umesto `pg_query_params()`, što dozvoljava stacked queries.
- **Zašto:** Kontrolišemo ceo sadržaj između navodnika; možemo zatvoriti string i dodati proizvoljne SQL naredbe.



**Kako sam potvrdio ranjivost**



1. Ulogovao sam se kao `user1` (nakon Step 1).
2. Poslao sam ručni POST zahtev s Burp Suite-om:

```
POST /class.php HTTP/1.1
...
title=test' ; SELECT pg_sleep(2) --
```

3. Odgovor je stigao ~2 sekunde kasnije → **potvrđen time-based SQLi**.
4. Proverio sam DB log:

```bash
docker logs tudo-db 2>&1 | tail -5
```

   - Vidljiv je SQL upit s ubačenom `pg_sleep` naredbom — injekcija izvršena.



**Exploit payload — stacked UPDATE**



- Payload za eskalaciju privilegija:

```
' ; UPDATE users SET is_admin=true WHERE username='user1' --
```

- Dekompozicija:
  - `'` — zatvara originalni string literal.
  - `;` — PostgreSQL separator koji dozvoljava novu naredbu.
  - `UPDATE users SET is_admin=true WHERE username='user1'` — direktna izmena privilegija.
  - ` --` — komentariše ostatak originalnog upita (sprečava SQL sintaksnu grešku).
- Rezultat u bazi:

```bash
docker exec -it tudo-db psql -U postgres tudo -c "SELECT username, is_admin FROM users;"
```

```
 username | is_admin
----------+----------
 admin    | t
 user1    | t        ← promenjen
 user2    | f
```



**Verifikacija — admin panel**



- Nakon injekcije, `GET /admin.php` vraća HTTP 200 (umesto 302 redirect) → korisnik `user1` je sada admin.
- **Zašto:** Aplikacija proverava `is_admin` iz baze pri svakom zahtevu; posle UPDATE-a, provera prolazi.



**Exploit skripta**



- Lokacija: `tudo/scripts/privesc.py` (helper), `tudo/scripts/exploit_chain.py` (Step 2).
- Pokretanje (Step 1 + Step 2 zajedno):

```bash
cd tudo
python3 scripts/exploit_chain.py http://localhost:8000 user1 'nova_sifra'
```

- Koraci koje skripta radi (Step 2):
  1. Proverava da li je korisnik već admin (`GET /admin.php`).
  2. Šalje `POST /class.php` s malicioznim `title` parametrom (stacked UPDATE).
  3. Ponovo proverava `/admin.php` — ako vraća 200, privesc je uspeo.



**Rezime privesc provere**



- Tip ranjivosti: SQL Injection (stacked queries) — CWE-89.
- Mesto: `app/class.php`, parametar `$_POST['title']`, funkcija `pg_query()`.
- Impact: autentifikovani korisnik bez administratorskih prava može sebi dodeliti `is_admin=true` i dobiti pun pristup admin panelu.
- PoC: `privesc.py` pozvan iz `exploit_chain.py` Step 2.



**Kako sprečiti ovu ranjivost**



- Koristiti **parameterizovane upite** (`pg_query_params()`) umesto spajanja stringa:

```php
// Sigurno:
$result = pg_query_params($conn,
    "SELECT * FROM class_posts WHERE title = $1",
    [$_POST['title']]
);
```

- `pg_query_params()` tretira parametre kao **podatke**, a ne kao deo SQL sintakse — stacked queries i string escape nisu mogući.
- Primeniti **principe najmanjeg privilegija** na nivou DB korisnika: aplikacioni DB nalog ne bi trebalo da ima `UPDATE` dozvolu na tabeli `users`.
- Dodati serversku validaciju ulaza (whitelist karaktera, maksimalna dužina) kao dodatni sloj odbrane.
- **Zašto:** Parametrizacija eliminiše SQLi na samom izvoru — napadač ne može izaći iz konteksta podatka bez obzira na sadržaj unosa.


**Remote Code Execution - Ime Prezime SVXX/2022**