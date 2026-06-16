# Oblak - kostur dokumentacije

## 1. Uvod

Oblak je serverless platforma za izvršavanje korisničkog Python koda na serveru. Ideja sistema je slična servisima kao što su AWS Lambda ili Google Cloud Functions: korisnik kroz CLI šalje kod, server ga proverava, priprema za izvršavanje, objavljuje funkciju i omogućava pokretanje preko generisanog HTTP endpoint-a.

Ovaj dokument predstavlja početni kostur projektne dokumentacije. Zasniva se na projektnom uputstvu iz fajla `Oblak.md` i postojećim tehničkim dokumentima u repozitorijumu:

- `README.md`
- `RUNNER_SYSTEM.md`
- `FIRECRACKER_ARCHITECTURE.md`
- `WSL2_FIRECRACKER_SETUP.md`
- `SETUP_CHECKLIST.md`
- `IMPLEMENTATION_SUMMARY.md`
- `ENABLE_FIRECRACKER.md`
- `QUICK_START.md`

Poseban fokus dokumentacije je na bezbednom izvršavanju potencijalno nebezbednog koda u izolovanom okruženju, odnosno u Firecracker microVM virtuelnoj mašini.

## 2. Ciljevi sistema

Glavni cilj sistema je da omogući kontrolisano izvršavanje korisničkih Python funkcija, uz proveru koda pre objavljivanja i izolaciju prilikom izvršavanja.

Očekivane funkcionalnosti iz uputstva:

- autentikacija CLI aplikacije prema serveru
- prenos Python koda na server, uključujući `requirements.txt` kada postoji
- analiza prenetog koda kroz strukturnu proveru, antivirus, statičku analizu i potencijalno LLM analizu
- priprema koda za izvršavanje, uključujući zavisnosti
- kreiranje URL-a za pokretanje objavljene funkcije
- pokretanje funkcije na zahtev
- izvršavanje nepoverljivog koda u izolovanom okruženju, preporučeno kroz Firecracker

## 3. Opis sistema

Sistem se sastoji od sledećih celina:

- API server implementiran u Go jeziku
- CLI alat za rad sa platformom
- skladište funkcija i verzija funkcija na disku
- verifikacioni pipeline za proveru poslatog koda
- runner sloj za izvršavanje funkcija
- baza podataka za korisnike, funkcije, verzije, deploy statuse i run rezultate
- lokalni runner za razvoj i testiranje
- Firecracker runner za izolovano izvršavanje u microVM okruženju

Osnovni tok rada:

```text
Korisnik / CLI
    |
    | login, deploy, publish, invoke
    v
API server
    |
    | upload funkcije
    v
Verifier pipeline
    |
    | verified / rejected
    v
Deploy / publish
    |
    | POST /invoke/{function_id}
    v
Runner interface
    |
    +-- Local runner
    |
    +-- Firecracker runner
            |
            v
        microVM + guest agent + Python funkcija
```

## 4. API i CLI tokovi

### 4.1 Autentikacija

CLI se autentikuje prema API serveru preko login endpoint-a. Server vraća token koji se koristi za dalje pozive.

Relevantni endpoint-i:

- `POST /auth/login`
- `GET /me`

Podrazumevani razvojni korisnik:

```text
user: admin
pass: admin
```

Za produkcionu upotrebu ovo mora biti zamenjeno bezbednijim mehanizmom upravljanja korisnicima i tajnama.

### 4.2 Upload funkcije

Korisnik šalje Python funkciju kao paket. Minimalna očekivana struktura funkcije je:

```text
main.py
requirements.txt   # opciono
```

Nakon prijema, server smešta izvornu arhivu i raspakovani radni direktorijum u `storage/functions/...`.

### 4.3 Verifikacija koda

Nakon upload-a pokreće se verifikacioni pipeline:

1. `structural_av` proverava strukturu arhive, ekstenzije, path traversal pokušaje, magic bytes i prisustvo očekivanih fajlova.
2. `clamav` opciono skenira raspakovani sadržaj pomoću ClamAV-a.
3. `static_bandit` pokreće Bandit statičku analizu Python koda.

Verzija funkcije dobija status:

- `verified` ako je prošla provere
- `rejected` ako je odbijena

Objavljivanje je dozvoljeno samo za verifikovane verzije.

### 4.4 Publish / deploy

Deploy endpoint objavljuje verifikovanu verziju funkcije i čini je dostupnom za pozivanje.

Relevantni endpoint:

```text
POST /functions/{id}/deploy
```

### 4.5 Invoke

Pokretanje funkcije vrši se preko:

```text
POST /invoke/{function_id}
```

Server učitava aktivnu verziju funkcije, generiše run ID, poziva odgovarajući runner, snima rezultat izvršavanja i vraća JSON odgovor.

Tipičan rezultat izvršavanja:

```json
{
  "run_id": "...",
  "function_id": "...",
  "version_id": "...",
  "status": "done",
  "exit_code": 0,
  "stdout": "...",
  "stderr": ""
}
```

## 5. Runner arhitektura

Runner sloj je centralna apstrakcija za izvršavanje funkcija. Definisan je interfejsom:

```go
type Runner interface {
    Invoke(ctx context.Context, req InvokeRequest) (*InvokeResult, error)
    Close() error
}
```

Ovaj pristup omogućava da server ne zavisi direktno od konkretnog načina izvršavanja. Trenutno postoje dva glavna backend-a:

- Local runner
- Firecracker runner

### 5.1 Local runner

Local runner pokreće `main.py` kao lokalni subprocess na host mašini. Koristi se za razvoj i brze testove.

Karakteristike:

- veoma brzo pokretanje
- jednostavno za razvoj
- radi na Windows, Linux i macOS sistemima
- nema izolaciju od host sistema
- nije pogodan za izvršavanje nepoverljivog koda

### 5.2 Firecracker runner

Firecracker runner pokreće funkciju unutar lake virtuelne mašine. Svako izvršavanje dobija sopstveni radni direktorijum, kopiju rootfs slike i poseban Firecracker proces.

Karakteristike:

- izolacija na nivou virtuelne mašine
- mogućnost ograničavanja CPU i memorije
- komunikacija hosta i guest-a preko vsock-a
- keširanje read-only ext4 slike funkcije po verziji, kako se ista funkcija ne bi pakovala pri svakom pozivu
- veći cold start u odnosu na lokalno izvršavanje
- zahteva Linux/KVM ili WSL2 sa nested virtualization podrškom

## 6. Firecracker izvršavanje

Firecracker je VMM namenjen pokretanju microVM-ova. U ovom projektu služi kao izolacioni sloj za korisnički Python kod.

Preduslovi:

- Linux ili WSL2
- dostupan `/dev/kvm`
- instaliran `firecracker` binarni fajl
- kompatibilna Linux kernel slika
- rootfs slika sa Python-om i guest agent-om

Konfiguracija se aktivira preko promenljivih okruženja:

```bash
export FIRECRACKER_KERNEL=/path/to/vmlinux
export FIRECRACKER_ROOTFS=/path/to/rootfs.ext4
```

Ako su ove promenljive postavljene, server pokušava da inicijalizuje Firecracker runner. Ako inicijalizacija ne uspe, sistem može da se vrati na Local runner za razvojni režim.

### 6.1 Životni ciklus jednog izvršavanja

Tok izvršavanja kroz Firecracker runner:

1. API primi `POST /invoke/{function_id}` zahtev.
2. Handler pronalazi aktivnu verziju funkcije.
3. Server formira `InvokeRequest`.
4. Firecracker runner generiše jedinstveni run ID.
5. Kreira se privremeni run direktorijum.
6. Rootfs slika se kopira kako bi VM imala izolovanu writable kopiju.
7. Proverava se cached ext4 image funkcije za dati `function_id/version_id`; ako ne postoji, radni direktorijum funkcije se pretvara u ext4 image i čuva u cache direktorijumu.
8. Pokreće se Firecracker proces sa Unix API socket-om.
9. Preko Firecracker API-ja podešavaju se:
   - broj vCPU jezgara
   - memorija
   - kernel image
   - boot argumenti, uključujući `root=/dev/vda`
   - rootfs disk
   - disk sa funkcijom
   - vsock uređaj
10. VM se startuje.
11. Host kroz retry mehanizam čeka da guest agent počne da sluša na vsock portu.
12. Host šalje JSON zahtev sa `function_id`, `version_id` i payload-om.
13. Guest agent pokreće `/function/main.py`.
14. Guest agent vraća `exit_code`, `stdout` i `stderr`.
15. Server snima rezultat u bazu.
16. VM se zaustavlja i privremeni direktorijum se briše.

### 6.2 Komunikacija host-guest

Komunikacija se obavlja preko vsock mehanizma. Host šalje JSON zahtev:

```json
{
  "function_id": "fn-123",
  "version_id": "v1",
  "payload": "ulazni podaci"
}
```

Guest agent odgovara:

```json
{
  "exit_code": 0,
  "stdout": "rezultat funkcije",
  "stderr": ""
}
```

### 6.3 Guest agent

Guest agent je Python proces koji se pokreće unutar VM-a. Njegove odgovornosti su:

- sluša vsock port
- prima JSON zahteve od hosta
- proverava da li postoje `function_id` i `version_id`
- montira disk sa funkcijom kao read-only ako već nije montiran
- pokreće `main.py`
- prosleđuje payload kroz stdin
- hvata stdout, stderr i exit code
- vraća rezultat hostu
- beleži događaje u log

### 6.4 Sažeta analiza izvršavanja koda u VM-u

Korisnički kod je potencijalno nebezbedan i ne sme se izvršavati direktno nad host sistemom u produkcionom režimu. Firecracker smanjuje rizik tako što se kod pokreće u posebnoj microVM instanci, sa sopstvenim kernel kontekstom, rootfs slikom i ograničenim komunikacionim kanalom ka hostu.

Najvažnije bezbednosne osobine:

- funkcija nema direktan pristup host fajl sistemu
- kod funkcije se montira kao read-only disk
- rootfs se kopira po izvršavanju, čime se smanjuje rizik trajne kontaminacije okruženja
- komunikacija sa hostom prolazi kroz kontrolisani JSON protokol preko vsock-a
- host dobija samo izlazne podatke: exit code, stdout i stderr
- timeout u guest agent-u sprečava beskonačno izvršavanje funkcije
- cached image funkcije je read-only u VM-u, pa ubrzava ponovljena izvršavanja iste verzije bez davanja write pristupa korisničkom kodu

Preostali rizici i otvorene stavke:

- potrebno je dodatno ograničiti resurse po izvršavanju
- potrebno je uvesti strožu mrežnu politiku unutar VM-a
- guest rootfs mora biti minimalan i redovno ažuriran
- treba razmotriti seccomp i jailer konfiguraciju Firecracker-a
- potrebno je proveriti da li stdout/stderr mogu sadržati osetljive podatke
- treba ograničiti veličinu payload-a i veličinu rezultata
- potrebno je detaljno dokumentovati cleanup u slučaju pada Firecracker procesa

## 7. Bezbednosni zahtevi

Bezbednosni zahtevi sistema:

- samo autentikovani korisnici smeju da šalju, objavljuju i pokreću funkcije
- nevalidan ili maliciozan paket mora biti odbijen pre deploy faze
- publish mora biti dozvoljen samo za proverene verzije
- izvršavanje nepoverljivog koda mora biti izolovano od hosta
- sistem mora beležiti upload, verifikaciju, deploy i invoke događaje
- rezultati izvršavanja moraju biti povezani sa korisnikom, funkcijom, verzijom i run ID-jem
- funkcija ne sme dobiti nekontrolisan pristup host fajl sistemu
- funkcija ne sme moći da iscrpi CPU, memoriju ili disk bez ograničenja
- greške moraju biti dovoljno opisne za reviziju, ali ne smeju nepotrebno otkrivati tajne
- svi bezbednosni alati i odluke moraju biti dokumentovani

## 8. Model pretnji

### 8.1 Akteri

- legitimni korisnik platforme
- napadač bez naloga
- napadač sa validnim nalogom
- maliciozna funkcija poslata kroz CLI
- kompromitovan ili ranjiv dependency
- administrator sistema

### 8.2 Vrednosti koje se štite

- host sistem
- baza podataka
- tokeni i kredencijali
- izvorni kod funkcija
- rezultati izvršavanja
- integritet verifier pipeline-a
- Firecracker kernel i rootfs slike
- logovi i audit tragovi

### 8.3 Granice poverenja

Granice poverenja postoje između:

- korisnika i API servera
- CLI alata i mreže
- upload arhive i verifier-a
- verifier-a i deploy faze
- API servera i runner-a
- hosta i Firecracker VM-a
- guest agent-a i korisničke funkcije
- aplikacije i baze podataka

## 9. STRIDE analiza

Ovo poglavlje treba proširiti u finalnoj dokumentaciji. Početna matrica:

| STRIDE kategorija | Primer pretnje | Postojeća mera | Otvorena stavka |
|---|---|---|---|
| Spoofing | Lažno predstavljanje korisnika prema API-ju | Login i bearer token | Ojačati upravljanje tokenima i rotaciju |
| Tampering | Izmena arhive kroz path traversal | Structural AV provera | Dodati detaljne testove arhiva |
| Repudiation | Korisnik poriče da je pokrenuo funkciju | Run rezultat i run ID | Proširiti audit log sa korisnikom i IP adresom |
| Information disclosure | Funkcija ispisuje tajne u stdout | Snimanje stdout/stderr za rezultat | Uvesti masking ili politiku za tajne |
| Denial of service | Beskonačna petlja ili ogroman output | Timeout u guest agent-u | Ograničiti memoriju, output i broj paralelnih VM-ova |
| Elevation of privilege | Bekstvo iz sandbox-a ka hostu | Firecracker VM izolacija | Uvesti jailer/seccomp/AppArmor gde je primenljivo |

## 10. Revizija i audit

Sistem treba da omogući naknadnu proveru bitnih događaja:

- login pokušaji
- upload funkcije
- rezultat verifikacije
- publish/deploy odluke
- svaki invoke zahtev
- run ID, funkcija, verzija, vreme izvršavanja
- exit code, stdout, stderr i greška
- izbor runner-a
- Firecracker inicijalizacija i greške

Otvoreno za dopunu:

- definisati format audit loga
- definisati retention politiku
- definisati šta se sme čuvati u stdout/stderr
- dodati korelacioni ID kroz ceo request flow

## 11. Statička analiza i skeniranje softvera

Za korisnički Python kod koristi se Bandit. Za antivirus skeniranje opciono se koristi ClamAV. Za Go kod projekta treba navesti i izvršiti dodatne alate.

Predloženi alati:

- `go test ./...`
- `go vet ./...`
- `gosec ./...`
- Bandit za Python sample funkcije
- ClamAV za upload pakete

U finalnoj dokumentaciji treba dodati rezultate pokretanja alata i nalaze code review-a.

## 12. Test primeri

Postojeći primeri iz projekta:

| Primer | Očekivani rezultat |
|---|---|
| `samples/benign/hello_world` | `verified` |
| `samples/malicious/subprocess_shell` | `rejected` kroz Bandit |
| `samples/malicious/eval_exec` | `rejected` kroz Bandit |
| `samples/malicious/missing_main` | `rejected` kroz structural proveru |
| `samples/malicious/forbidden_script` | `rejected` kroz structural proveru |
| `samples/malicious/nested_main` | `rejected` kroz structural proveru |

Dodatni testovi koje treba dokumentovati:

- benigni invoke kroz Local runner
- benigni invoke kroz Firecracker runner
- funkcija koja baca exception
- funkcija koja piše na stderr
- funkcija koja traje duže od timeout-a
- funkcija koja pokušava pristup host putanjama
- funkcija koja pokušava mrežni pristup
- funkcija koja pravi veliki output
- funkcija sa malicioznim dependency-jem

## 13. Uputstvo za pokretanje

### 13.1 Pokretanje servera sa Firecracker runner-om iz Windows-a

Ako je Firecracker okruženje pripremljeno u WSL2 distribuciji `Ubuntu-24.04`, server se može pokrenuti direktno iz Windows terminala komandom:

```powershell
wsl -d Ubuntu-24.04 -e bash -c "cd /mnt/c/Users/Aleksa/Desktop/rbs/projekat && bash run-firecracker.sh"
```

Ova komanda ulazi u WSL distribuciju, prelazi u projektni direktorijum na Windows disku i pokreće `run-firecracker.sh`.

Skripta radi sledeće:

- proverava da li je `firecracker` dostupan u `PATH`
- proverava da li postoje kernel i rootfs u `~/firecracker-workspace`
- proverava dostupnost `/dev/kvm`
- kopira `vmlinux` i `rootfs.ext4` u projektni `storage`
- proverava ili lokalno priprema Go
- postavlja `FIRECRACKER_KERNEL`, `FIRECRACKER_ROOTFS`, `OBLAK_RUNS_DIR`, `OBLAK_ADDR` i `OBLAK_DB`
- pokreće API server komandom `go run ./cmd/api/main.go`

Očekivani server endpoint-i nakon pokretanja:

```text
API: http://127.0.0.1:8080
UI:  http://127.0.0.1:8080/ui
```

### 13.2 Preduslovi za WSL2 pokretanje

Pre pokretanja skripte potrebno je da važi:

- instaliran je WSL2 sa distribucijom `Ubuntu-24.04`
- u `.wslconfig` je uključeno `nestedVirtualization=true`
- WSL je restartovan komandom `wsl --shutdown`
- u Ubuntu okruženju postoji `/dev/kvm`
- korisnik ima dozvolu za `/dev/kvm` ili je dodat u `kvm` grupu
- Firecracker binarni fajl je instaliran
- `~/firecracker-workspace/vmlinux` postoji
- `~/firecracker-workspace/rootfs.ext4` postoji
- rootfs sadrži Python 3 i guest agent

Brza provera iz Windows-a:

```powershell
wsl -d Ubuntu-24.04 -e bash -c "ls -l /dev/kvm && firecracker --version && ls -lh ~/firecracker-workspace/vmlinux ~/firecracker-workspace/rootfs.ext4"
```

### 13.3 Pokretanje lokalnog runner-a

Za razvoj bez Firecracker-a dovoljno je pokrenuti API bez `FIRECRACKER_KERNEL` i `FIRECRACKER_ROOTFS` promenljivih:

```powershell
go run ./cmd/api/main.go
```

U tom režimu sistem koristi Local runner i pokreće `main.py` kao subprocess na hostu. Ovaj režim je praktičan za razvoj, ali nije bezbedan za nepoverljiv kod.

### 13.4 Testiranje nakon pokretanja

Tipičan tok testiranja:

```powershell
go run ./cmd/cli login --user admin --pass admin
go run ./cmd/cli deploy --path .\samples\benign\hello_world --name hello_world
go run ./cmd/cli publish <function_id>
go run ./cmd/cli invoke <function_id>
```

Kod Firecracker runner-a prvi poziv iste verzije funkcije pravi cached ext4 image funkcije. Naredni pozivi iste verzije koriste postojeći cached image, ali i dalje pokreću novu microVM instancu, pa cold boot trošak i dalje postoji.

### 13.5 Napomene o performansama

Trenutno je optimizovan deo pripreme funkcijskog diska:

- `function.ext4` se cache-uje po `function_id/version_id`
- disk funkcije se montira read-only u guest-u
- rootfs se i dalje kopira po svakom izvršavanju
- Firecracker VM se i dalje pokreće iznova po svakom invoke zahtevu

Zbog toga ponovljeni pozivi mogu biti brži u delu pripreme image-a, ali neće biti "warm" u punom serverless smislu. Za značajno ubrzanje potrebni su warm VM pool, snapshot/restore ili direktniji init bez punog systemd boot-a.

## 14. Ograničenja trenutne implementacije

Trenutna implementacija predstavlja funkcionalan skeleton platforme, ali određeni delovi zahtevaju dodatno ojačavanje:

- Local runner nema bezbednosnu izolaciju
- Firecracker setup zavisi od Linux/KVM okruženja
- rootfs i kernel moraju biti ručno pripremljeni
- funkcijski ext4 image se cache-uje po verziji, ali VM se i dalje pokreće iznova za svaki invoke
- zavisnosti iz `requirements.txt` treba preciznije dokumentovati i izolovati
- LLM analiza je navedena kao mogućnost, ali nije obavezni deo trenutne implementacije
- monitoring, rate limiting i kvote treba dodati kao otvorene stavke
- potrebno je proširiti reviziju i audit događaje
- potrebno je proveriti ponašanje pri paralelnim izvršavanjima

## 15. Predlog strukture finalne dokumentacije

Finalni dokument može biti organizovan ovako:

1. Uvod i cilj projekta
2. Funkcionalni zahtevi
3. Arhitektura sistema
4. API i CLI tokovi
5. Skladištenje funkcija i verzionisanje
6. Verifikacioni pipeline
7. Runner sistem
8. Firecracker arhitektura
9. Analiza izvršavanja koda u VM-u
10. Bezbednosni zahtevi
11. Model pretnji
12. STRIDE analiza
13. Implementirane mere ublažavanja
14. Otvorene bezbednosne stavke
15. Revizija, audit i logovanje
16. Statička analiza i rezultati alata
17. Code review proces
18. Test primeri, benigni i maliciozni
19. Uputstvo za pokretanje
20. Firecracker performanse i cache funkcijskih image-a
21. Zaključak

## 16. Zaključak

Oblak platforma već ima jasnu podelu na API, CLI, verifier i runner sloj. Najvažniji bezbednosni mehanizam za produkciono izvršavanje je Firecracker microVM, jer odvaja korisnički kod od host sistema i svodi komunikaciju na kontrolisani kanal preko guest agent-a. Finalnu dokumentaciju treba proširiti detaljnim STRIDE modelom, rezultatima statičke analize, opisom audit mehanizama i dokazima kroz benigno i maliciozno testiranje.
