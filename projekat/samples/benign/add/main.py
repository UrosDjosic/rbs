import json
import sys


def handler(event):
    a = event.get("a", 0)
    b = event.get("b", 0)
    return {"result": a + b}


if __name__ == "__main__":
    raw = sys.stdin.read().strip()
    if not raw:
        print(json.dumps({"error": "missing input"}))
        sys.exit(1)
    event = json.loads(raw)
    print(json.dumps(handler(event)))
