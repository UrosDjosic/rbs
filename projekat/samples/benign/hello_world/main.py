import json
import sys


def handler(event=None):
    return {"message": "hello from oblak", "event": event}


if __name__ == "__main__":
    event = {"from": "cli"}
    raw = sys.stdin.read().strip()
    if raw:
        event = json.loads(raw)
    print(json.dumps(handler(event)))

