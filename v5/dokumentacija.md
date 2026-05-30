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


**Admin Privilege Escalation - Ime Prezime SVXX/2022**


**Remote Code Execution - Ime Prezime SVXX/2022**