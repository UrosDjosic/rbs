THREAT MODELING

**1\. MOTIVACIJA NAPADAČA**

**Sajber kriminalci \*\*\***

- **Motivacija:** Finansijska korist
- **Opis:** Pojedinci ili organizovane grupe koje žele profit.
- **Nivo veštine:** Srednji / visok
- **Pristup sistemu:** kompromitovani nalozi, eksterni pristup preko interneta putem web aplikacije, API servisa, phishing napada nad zaposlenim
- **Krajnji cilj:** Krađa podataka o platnim karticama, krađa naloga, prodaja ličnih podataka, lažna plaćanja, ransomware.

**Rivalske kompanije \*\*\***

- **Motivacija:** Indirektna finansija korist, prosperiranje svoje kompanije
- **Opis:** Kompanije u turizmu koje žele da steknu poslovnu prednost
- **Nivo veštine:** visok
- **Pristup sistemu:** kompromitovani zaposleni, phishing napad, iskorišćavanje ranjivosti javnih servisa
- **Krajnji cilj:** krađa korisnika I poslovnih strategija, cene nekih određenih ugovora, narušavanje reputacije

**Insajderi (zaposleni / bivši zaposleni) \*\*\***

- **Motivacija:** Osveta firmi, finansijska korist
- **Opis:** Mogu biti plaćenici od strane drugih kompanija, mogu raditi u svoju finansijsku korist ili imaju neku vrstu vendete prema kompaniji
- **Nivo veštine:** nizak/srednji/visok
- **Pristup sistemu:** legitiman interni pristup, imaju privilegije ulaska I rukovodstva internih poslovnih sistema
- **Krajnji cilj:** krađa ličnih podataka, krađa platnih detalja, prodaja internih informacija, sabotaža sistema

**Hacktivisti**

- **Motivacija:** Protest protiv kompanije
- **Opis:** Grupe sa političkim ciljevima
- **Nivo veštine:** srednji
- **Pristup sistemu:** eksterni pristup, iskorišćavanje ranjihovosti javnih servisa, DDoS napadi
- **Krajnji cilj:** rušenje sajta, curenje podataka

**Oportunisti \*\*\***

- **Motivacija:** Zabava uglavnom
- **Opis:** neiskusni uglavnom neplaćeni napadači
- **Nivo veštine:** nizak
- **Pristup sistemu:** korišćenje automatizovanih alata, pokušaji prijave na korisničke naloge (ili naloge zaspolenih)
- **Krajnji cilj:** testiranje ranjivosti, brute force login napadi

**Državno podržani napadači**

- **Motivacija:** Nadzor građana (mogu biti političari, poslovni ljudi, ali uglavnom uticajni), prikupljanje njima bitnih podataka
- **Opis:** Napredne grupe koje rade u korist neke države
- **Nivo veštine:** visok
- **Pristup sistemu:** napredan eksterni pristup, spear phishing napadi, zero-day ranjivosti, supply chain attack, kompromitovani zaposleni pa onda I interni pristup
- **Krajnji cilj:** prikupljanje podataka

**2\. IMOVINA**

**1\. Baza korisničkih naloga**

**• Izloženost (ko ima pristup):** korisnici aplikacije, administratori sistema, baze podataka, backend servisi

**• CIA ciljevi:** zaštita korisničkih podataka I lozinki, tačnost podataka naloga I mogućnost prijave korisnika u svakom trenutku

**• Uticaj kompromitacije:** krađa naloga, neovlašćen pristup, gubitak poverenja korisnika I samim tim reputaciona I finansijska šteta

**2\. Podaci o platnim karticama / transakcijama**

**• Izloženost (ko ima pristup):** korišćeni platni sistemi, određeni administratori, backend servisi za obradu plaćanja

**• CIA ciljevi:** zaštita podataka kartica I transakcija, tačnost iznosa I istorije tranzakcija, nesmetana obrada plaćanja

**• Uticaj kompromitacije:** finansijska krađa, zloupotreba kartica, prekid plaćanja, reputaciona I finansijska šteta

**3\. Rezervacije putovanja i istorija putovanja**

**• Izloženost (ko ima pristup):** korisnici, zaspoleni korisničke podrške, backend servisi, partneri kompanije poput hotela, avio kompanija itd.

**• CIA ciljevi:** zaštita podataka o putovanjima korisnika, tačnost rezervacija I statusa putovanja, mogućnost upravljanja rezervacijama u svakom trenutku

**• Uticaj kompromitacije:** neželjeno otkazivanje ili izmena rezervacije, curenje prihvatnih podataka, samim tim nezadovoljstvo korisnika, reputaciona I finansijska šteta

**4\. Lični podaci korisnika (PII)**

**• Izloženost (ko ima pristup):** korisnici svojih naloga, ovlašćeni zasposleni I administratori sistema, backend servisi, parterni kojima su podaci neophodni za uslugu koju pružaju

**• CIA ciljevi:** zaštita, ličnih podataka, tačnost korisničkih podataka, dostupnost za obradu I pružanje usluga

**• Uticaj kompromitacije:** krađa identiteta, zloupotreba ličnih podataka, zakonske kazne, gubitak poverenja, reputaciona I finansijska šteta

**5\. Admin nalozi i privilegovani pristupi**

**• Izloženost (ko ima pristup):** sistem administratori, bezbednosni tim, ovlašćeni zaposleni sa visokim privilegijama, DevOps tim

**• CIA ciljevi:** zaštita admin privilegija I pristupa, sprečavanje neovlašćenih izmena podataka, dostupnost admin naloga za hitne slučajeve

**• Uticaj kompromitacije:** potpuna kontrola nad sistemom, sve od prethodnog navedeno

**6\. Web aplikacija**

**• Izloženost (ko ima pristup):** Korisnici putem interneta, zaposleni, povezani backend servisi, administratori

**• CIA ciljevi:** zaštita korisničkih sesija, podataka I komunikacije, ispravno funkcionisanje aplikacije, neprekidan rad I pristup aplikacije

**• Uticaj kompromitacije:** nedostupnost sajta, krađa naloga, neovlašćene izmene prikazanih podataka, finansijska I reputaciona šteta

**7\. API servisi**

**• Izloženost (ko ima pristup):** Web aplikacija, mobilna aplikacija (ukoliko postoji), interni sistemi kompanije I ovlašćeni sistemi putem interneta (hotelski sistemi, payment sistemi itd.)

**• CIA ciljevi:** zaštita podataka koji se razmenjuju putem API poziva, tačnost zahteva, odgovora I transkacija, neprekidan rad servisa

**• Uticaj kompromitacije:** neovlašćen pristup podacima, prekid rada, lažni zahtevi, finansijska I reputaciona šteta

**8\. Baza partnera (hoteli, avio kompanije, rent-a-car)**

**• Izloženost (ko ima pristup):** ovlašćeni zaposleni, administratori sistema, interni servisi kompanije i partnerski sistemi

**• CIA ciljevi:** zaštita podataka o partnerima, ugovorima, cenama, tačnost ovih podataka I njihova stalna dostupnost radi rezervacija I poslovne saradnje

**• Uticaj kompromitacije:** curenje poverljivih poslovnih informacija, prekid saradnje sa partnerima, pogrešne rezervacije I zahtevi, finansijska I reputaciona šteta

**9\. Interna dokumentacija i poslovne strategije**

**• Izloženost (ko ima pristup):** menadžment, ovlašćeni zaposleni, administratori sistema i interni poslovni sistemi kompanije

**• CIA ciljevi:** zaštita poslovnih planova, procedura I finansijskih informacija, tačnost I neizmenjenost internih dokumenata I njihova dostupnost

**• Uticaj kompromitacije:** gubitak neke ostvarene poslovne prednosti, curenje poverljivih informacija, reputaciona I finansijska šteta

**10\. Serveri / cloud infrastruktura**

**• Izloženost (ko ima pristup):** administratori sistema, tehničko osoblje, ovlašćeni interni servisi, cloud provider

**• CIA ciljevi:** zaštita konfiguracije sistema, podataka I ključeva, ispravno funkcionisanje servera, mreže I podešavanja I njihov neprekidan rad

**• Uticaj kompromitacije:** pad sistema, gubitak podataka, finansijska I reputaciona šteta  
<br/>

**3\. POVRŠINA NAPADA**

Korisnici koji komuniciraju sa sistemom jesu:

- Klijenti (putnici) : veliki broj putnika (merljiv u milionima) pristupa MegaTravel platformi radi istraživanja i rezervacije usluga
- Zaposleni : Mnoštvo zaposlenih lica koje koriste aplikacije u više slojeva : npr admin, prodavac…
- Sistemi za smeštaj(partneri) : Eksterni sistemi hotela pomoću koje dobijamo informacije o dostupnosti i mestima u hotelima.
- Sistemi za prevoz(avio-kompanije) : Eksterni sistemi pomoću kojih pristupamo ponudama različitih avio kompanija koje su dostupne krajnjim korisnicima.
- Sistemi za obradu plaćanja (payment system) : Eksterni procesori za realizaciju transakcija (najčešće banke).

Mapiranje ulaznih tačaka napada na površinu:

- Javni digitalni kanali, kao što su veb/mobilna aplikacija koje predstavlja glavnu tačku interfejsa klijenta.
- Korporativni sistemi - u to spadaju administritavni paneli koje koriste zaposleni za rukovanje nalozima i resursima, email serveri koje služe za komunikaciju sa klijentima. Ovde se pojavljuje pojam „phishing" kao učestao i uspešan način za distribuciju zlonamernog softvera ili drugih pokušaja krađe putem lažnog mejla. (npr poruke koje stižu sa nepoznatih brojeva sa zahtevima poznatim ljudima)
- Autentifikacioni sistemi, što predstavlja login za korisnike I zaposlene, gde može biti kritična tačka zbog slabih lozniki, neadekvatne autentifikacije ili slično.
- Payment integracija - gde imamo rizik zbog slanja ličnih podataka klijenta. Predstavlja ulaznu tačku gde se može destiti krađa kartičnih podataka, manipulacija transakcijama I tako se kompormitovati ovaj ceo eksterni sistem.
- API integracije sa partnerima - Malopre pomenuti, takođe eksterni sistem, može predstavljati tačku ulaza. Rizici jesu nevalidirani ulazni podaci, zlonamerni partner ili kompromitovan API za komunikaciju.

**4\. DATA FLOW DIAGRAMS**

####1\. Kontekstni dijagram
![alt text](context_diagram.png) 

Na ovom dijagramu posmatramo samo MegaTravel kao svojstven sistem (backend, fontend, baze, infrastruktura). Posmatramo eksterne komunikacije I ko sve ima dodira sa našim sistemom, što nam ukazuje na potencijalne pretnje. Imamo četiri glavna podsistema - putnici, zaposleni, sistemi partnera (kao što su avio komapnije ili sistemi hotela za rezervacije) I sistemi za plaćanje (API banaka). Ovde možemo u širem kontekstu uočiti koje komunikacije se ostvaruju u našem sistemu I sa kojih pozicija se pretnja može pojaviti.

####2\. Dijagram toka podataka (ceo)
![alt text](<threats model.png>)


Na slici se nalazi data flow diagram u kome su uivičene granica poverenja, korporativna zona i kritična zona podataka. U suštini bi sva od ova tri mogla biti neke trust boundaries. Na jednoj strani imamo baze podataka i cloud infrastukturu, zatim web aplikaciju i autentifikacioni sistem. I kao treću celinu imamo korporativnu zonu koja obuhvata administratorske panele, poslovnu logiku, kao i API servise za komuniciranje za sistemima za plaćanje i partnerskim API-jima.

**5\. ANALIZA PRETNJI I MITIGACIJA**

<br/>