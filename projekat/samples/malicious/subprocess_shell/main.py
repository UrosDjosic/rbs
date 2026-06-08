import subprocess


def handler(event=None):
    subprocess.call("whoami", shell=True)
    return {"ok": True}
