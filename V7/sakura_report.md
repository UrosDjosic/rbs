# Sakura izazov write-up

TryHackMe soba [Sakura](https://tryhackme.com/room/sakura) (OSINT Dojo). Cilj je bio pratiti napadača kroz javno dostupne tragove, kao što su metapodaci fajlova, društvene mreže, GitHub istorija, blockchain i geolokacija fotografija.

---

## Task 2: TIPOFF

**Pitanje:** What username does the attacker go by?

**Odgovor:** `SakuraSnowAngelAiko`

Otvorili smo SVG sliku (`pwnedletter.svg`) koju je napadač ostavio. U izvornom kodu fajla, u metapodacima Inkscape alata, pronašli smo putanju izvoza:

```
inkscape:export-filename="/home/SakuraSnowAngelAiko/Desktop/pwnedletter.png"
```

Na Linux sistemima `/home/<ime>/` odgovara korisničkom nalogu, odatle smo izvukli username.

![Analiza SVG metapodataka](./sakura_images/picture1.png)

---

## Task 3: RECONNAISSANCE


**Pitanje:** What is the attacker's full real name?

**Odgovor:** `Aiko Abe`

Pretragom username-a `SakuraSnowAngelAiko` pronašli smo stare profile napadača na Twitteru i LinkedInu, gde je navedeno puno ime.


**Pitanje:** What is the full email address used by the attacker?

**Odgovor:** `SakuraSnowAngel83@protonmail.com`

Na GitHub nalogu `sakurasnowangelaiko` postoji repozitorijum `PGP` sa fajlom `publickey`. PGP javni ključ sadrži User ID, odnosno email adresu vlasnika ključa. Kopirali smo ceo PGP blok i dekodirali ga pomoću online alata ([pgptool.org](https://pgptool.org/) ili [keyserver.ubuntu.com](https://keyserver.ubuntu.com/)).

![PGP ključ na GitHubu](./sakura_images/pgp.png)

---

## Task 4: UNVEIL

**Pitanja i odgovori:**

| Pitanje | Odgovor |
|---------|---------|
| What cryptocurrency does the attacker own a wallet for? | `Ethereum` |
| What is the attacker's cryptocurrency wallet address? | `0xa102397dbeeBeFD8cD2F73A89122fCdB53abB6ef` |
| What mining pool did the attacker receive payments from on January 23, 2021 UTC? | `Ethermine` |
| What other cryptocurrency did the attacker exchange with? | `Tether` |

Clue je ukazivao da je napadač obrisao osetljive informacije, ali da ih možda možemo povratiti. Prva ideja bila je **GitHub commit istorija**, jer obrisani sadržaj često ostaje u prethodnim commit-ovima.

Pregledali smo repozitorijume na GitHubu i našli `ETH` repozitorijum sa fajlom `miningscript`. U istoriji commit-ova videli smo da je jedna linija zamenjena. U staroj verziji stajalo:

```
stratum://0xa102397dbeeBeFD8cD2F73A89122fCdB53abB6ef.Aiko:pswd@eu1.ethermine.org:4444
```

Ovo je mining konfiguracija za Ethereum (stratum protokol). Iz nje smo izvukli wallet adresu i mining pool (`eu1.ethermine.org` **Ethermine**). Wallet adresu smo zatim pretražili na [Etherscan](https://etherscan.io/) i tamo pronašli odgovore na preostala pitanja, uključujući razmenu sa **Tether (USDT)** tokenom.

![GitHub commit sa obrisanom mining konfiguracijom](./sakura_images/crypto.png)

![Etherscan, istorija transakcija](./sakura_images/crypto_history.png)

---

## Task 5: TAUNT


**Pitanje:** What is the attacker's current Twitter handle?

**Odgovor:** `SakuraLoverAiko`

Jednostavna Google pretraga za `SakuraSnowAngelAiko` vodi do Twitter profila napadača.


**Pitanje:** What is the BSSID for the attacker's Home WiFi?

**Odgovor:** `84:AF:EC:34:FC:F8`

U prvom koraku pronađen je screenshot sa dark web paste sajta **DeepPaste**, gde je napadač (anonimno) sačuvao listu WiFi mreža i lozinki. Među njima piše:

- **Home WiFi:** SSID `DK1F-G`
- **City Free WiFi:** SSID `HIROSAKI_FREE_Wi-Fi` (naziv ukazuje na grad Hirosaki u Japanu)

![DeepPaste, sačuvane WiFi lozinke](./sakura_images/deeppaste.png)

U drugom koraku SSID `DK1F-G` pretražili smo na [WiGLE.net](https://wigle.net/), bazi podataka koja mapira WiFi mreže (SSID i BSSID) na geografske koordinate na osnovu wardriving podataka. WiGLE je vratio jedan rezultat sa BSSID-om `84:AF:EC:34:FC:F8` i koordinatama u blizini Hirosakija, što potvrđuje da je ovo napadačeva kućna mreža.

![WiGLE pretraga za SSID DK1F-G](./sakura_images/wifi.png)

---

## Task 6: HOMEBOUND

### Najbliži aerodrom

**Pitanje:** What airport is closest to the location the attacker shared a photo from prior to getting on their flight?

**Odgovor:** `DCA`

Fotografiju sa Twitter profila analizirali smo pomoću **Google Lens**-a. AI je prepoznao Washington Monument okružen cvetajućim trešnjama, a lokacija je Washington D.C. / Arlington, Virginia. Zatim smo pretražili aerodrome u tom regionu i isprobali trocifrene kodove. Najbliži aerodrom centru grada je **Ronald Reagan Washington National Airport**, sa kodom **DCA**.

![Google Lens, Washington Monument](./sakura_images/washington.png)

![Pretraga aerodroma u Washington D.C.](./sakura_images/airport.png)

### Poslednji layover

**Pitanje:** What airport did the attacker have their last layover in?

**Odgovor:** `HND`

Druga fotografija sa Twittera prikazuje Japan Airlines (JAL) First Class / Sakura Lounge. Google Lens je identifikovao lounge i dao rezultate za različite aerodrome. Kod **JAL** nije prošao, ali smo u rezultatima pronašli **HND** (Haneda Airport, Tokio), što je bio tačan odgovor.

![JAL Sakura Lounge](./sakura_images/japan_airlines.png)

### Jezero na mapi

**Pitanje:** What lake can be seen in the map shared by the attacker as they were on their final flight home?

**Odgovor:** `Lake Inawashiro`

Na Twitter profilu pronašli smo satelitsku sliku Japana. Analizom pomoću Google Lens-a AI je predložio Yamagata prefecture. Otvorili smo Google Maps, uporedili oblik jezera na slici sa mapom i identifikovali **Lake Inawashiro**.

![Identifikacija jezera na satelitskoj slici](./sakura_images/lake.png)

![Twitter slika Yamagata](./sakura_images/yamagata.png)


### Kućni grad

**Pitanje:** What city does the attacker likely consider "home"?

**Odgovor:** `Hirosaki`

U DeepPaste listi WiFi mreža naziv `HIROSAKI_FREE_Wi-Fi` direktno ukazuje na grad Hirosaki. WiGLE pretraga za kućni SSID `DK1F-G` dodatno potvrđuje lokaciju u tom regionu.

---



## Uradjeni TASK-ovi
![Uradjeno](./sakura_images/done.png)



