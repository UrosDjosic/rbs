SYSTEM REVIEW AUDIT SKRIPTA

**OPERATING SYSTEM VIKTOR SRBLJIN SV63/2022**

**Detekcija verzije operativnog sistema**

- cat /etc/debian_version, cat /etc/redhat-release, cat /etc/fedora-release ili lsb_release -a — da se utvrdi koji distro i verzija su instalirani.
- **Zašto:** zastarela ili nepodržana verzija OS-a znači da sistem možda više ne dobija security patch-eve i ostaje ranjiv na poznate probleme.

**Upozorenje o EOL/support statusu**

- nakon provere verzije skripta ispisuje podsetnik auditoru da proveri da li distro još ima podršku.
- **Zašto:** sama verzija nije dovoljna, važno je znati da li proizvođač i dalje izdaje ispravke za taj sistem.

**KERNEL VIKTOR SRBLJIN SV63/2022**

**Puna informacija o kernelu (uname -a)**

- **Zašto:** daje verziju kernela, arhitekturu i hostname, osnova za proveru da li je kernel zastareo ili ranjiv.

**Pokrenuti kernel (uname -r)**

- **Zašto:** pokazuje koji kernel trenutno radi u memoriji, to je ono što sistem stvarno koristi, ne ono što je samo instalirano na disku.

**Instalirani kernel paketi (dpkg -l | grep linux-image)**

- **Zašto:** uporedba sa uname -r otkriva da li je noviji kernel instalovan ali nije pokrenut, patch je skinut, ali reboot nije urađen pa stari kernel i dalje radi.

**Poslednji boot (who -b)**

- **Zašto:** tačan datum poslednjeg paljenja mašine, pomaže da se uptime i kernel provere stave u kontekst.

**Uptime (uptime)**

- **Zašto:** ako sistem radi predugo bez restarta, verovatno nije bilo kernel reboot-a posle ažuriranja. Skripta daje [WARN] ako je uptime duži od 50 dana.

**Rezime kernel provere**

- na kraju se u par linija sumira šta treba uporediti: uname -r, dpkg linux-image pakete i uptime.
- **Zašto:** auditor brzo vidi da li je verovatno potreban reboot posle kernel update-a.



**TIME MANAGEMENT ALEKSA ŠILJIĆ SV50/2022**


### Provera vremenske zone (/etc/timezone, timedatectl)

- Skripta čita /etc/timezone, a ako fajl ne postoji, što najverovatnije znači da smo na novom sistemu, onda se koristi timedatectl kao fallback opcija.
- Ovo se radi zato što bi serveri trebalo ekskluzivno da koriste UTC zonu. Zone sa DST(letnje-zimsko računanje) uzrokuju skokove u logovima zbog pomeranja vremena unapred ili unazad. Kada se vraća unazad isto vreme se pokaže dva puta a kada se pomera unapred nedostaje sat vremena po logovima.

### Provera da li Network Time Protocol radi

- Skripta traži ntpd proces i ako ga nema proverava alternativne implementacije. `systemd-timesyncd` (uobičajeno na modernom Debianu) i `chrony`.

- Bez aktivnog NTP serivsa vreme na sistemu postepeno odstupa od tačnog. Posledice mogu biti : odbijanje zapravo validnog SSL/TLS sertifikata (svaki ima notBefore i notAfter polja), pad Kerberos autentifikacije (tolerancija je svega 5 minuta), i nemogućnost pouzdane kolekcije log zapisa sa više mašina pri istrazi potencijalnog incidenta.

### Provera NTP peer sinhronizacije (ntpq -p -n)

- Skripta pokreće ntpq -p -n i traži red koji počinje sa * - to označava aktivni izvor sinhronizacije. Na sistemima bez ntpq se koristi timedatectl kao fallback.
- Ovo se radi iz razloga što NTP daemn može biti pokrenut a da zapravo ne sinhronizuje vreme. Na primer kada firewall blokira UDP port 123 ka eksternim serverima. Bez ove provere greška ostaje "tiha", jer servis "radi", ali sta se ne usklađuje ni sa čim

- NTP rešava problem hardverskih lokalnih satova (kristalni oscilatori na matičnoj ploči) koji uglavnom jesu daleko od savršenih zbog starenja komponente, temperature... Rešava ovo tako što server pita periodično jedan ili više referentnih servera koliko je tačno vreme, izračuna razliku i koriguje svoje lokalno vreme. Ove male korekcije su poenta NTP protokola.

- Peer sinhronizacija se proverava kako npr. firewall ili slične greške se ne bi dešavale, odnosno da poslati paketi ne stignu do servera.

| Simbol | Remote          | Refid           | st | t | when | poll | reach | delay  | offset | jitter |
|--------|-----------------|-----------------|----|---|------|------|-------|--------|--------|--------|
| `*`    | 130.102.128.23  | 130.102.132.164 | 2  | u | 495  | 1024 | 377   | 59.171 | 0.984  | 2.292  |
| `+`    | 203.26.72.7     | 202.147.104.51  | 3  | u | 393  | 1024 | 377   | 58.965 | -0.952 | 9.207  |
| `-`    | 121.0.0.41      | 204.152.184.72  | 2  | u | 250  | 1024 | 377   | 54.635 | -2.797 | 0.877  |

- Primer komande `ntp -p -n` gde je zvezdica trenutan izvor komunikacije, + je kandidat, ali ne primarni peer. - je peer koji je alogritam odbacio kao nepouzdan, a x je peer koji je označen kao lažan, ignoriše se.


## **PACKAGES INSTALLED i LOGGING - Uros Djosic SV20/2022**

# **PACKAGES INSTALLED**

## **Broj instaliranih paketa (`dpkg -l | wc -l`)**

- prikazuje ukupan broj instaliranih paketa na sistemu.
- **Zašto:** veliki broj paketa povećava attack surface sistema i šansu da postoji zastareo ili ranjiv softver.

## **Lista instaliranih paketa (`dpkg -l`)**

- prikazuje pregled instaliranih paketa i njihovih verzija.
- **Zašto:** auditor može da identifikuje zastarele, nepotrebne ili potencijalno ranjive pakete.

## **Provera GUI / desktop paketa**

- koristi:

```bash
egrep 'gnome|kde|xfce|xorg|xserver'
```

- **Zašto:** serveri često ne zahtevaju grafičko okruženje. GUI povećava broj servisa, potrošnju resursa i attack surface.

## **Provera game paketa**

- koristi:

```bash
egrep 'game|steam|supertux'
```

- **Zašto:** igre i nepotrebni korisnički paketi nemaju mesto na produkcionom serveru i mogu predstavljati bezbednosni rizik.

## **Rezime package provere**

- skripta podseća da sistem treba da ima minimalan broj paketa potrebnih za svoju ulogu.
- **Zašto:** princip *minimal installation* smanjuje mogućnost kompromitovanja sistema.

---

# **LOGGING**

## **Provera syslog procesa (`ps -edf | grep syslog`)**

- proverava da li je syslog/rsyslog servis aktivan.
- **Zašto:** bez centralnog logging servisa bezbednosni događaji možda neće biti evidentirani.

## **Provera rsyslog konfiguracije (`/etc/rsyslog.conf`)**

- proveravaju se opcije:

  - `imudp`
  - `UDPServerRun`
  - `imtcp`
  - `InputTCPServerRun`

- **Zašto:** omogućeni UDP/TCP syslog receiver-i znače da sistem prima udaljene logove. Ako to nije potrebno, povećava se attack surface.

## **Provera permisija log fajlova**

- proveravaju se:

  - `FileCreateMode`
  - `DirCreateMode`
  - `Umask`

- **Zašto:** log fajlovi često sadrže osetljive informacije i ne smeju biti dostupni svim korisnicima.

## **Provera remote logging-a**

- proverava se da li postoji `@servername` konfiguracija u `rsyslog.conf`.

- **Zašto:** remote logging omogućava čuvanje logova na udaljenom sistemu. Ako napadač kompromituje server, teže može obrisati tragove sa udaljenog log servera.

## **Provera glavnih log fajlova**

- proverava postojanje:

  - `/var/log/syslog`
  - `/var/log/auth.log`

- **Zašto:** potvrđuje da sistem beleži opšte događaje i autentikacione pokušaje.

## **Rezime logging provere**

- skripta sumira ključne stvari koje auditor treba da proveri:

  - da logging servis radi,
  - da su permisije ograničene,
  - da li postoji remote logging,
  - da li su nepotrebni syslog receiver-i uključeni.

- **Zašto:** logging je ključan za detekciju incidenta, forenziku i audit aktivnosti na sistemu.

