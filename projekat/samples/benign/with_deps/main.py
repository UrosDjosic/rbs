import json
import sys

import colorama


def handler(event):
    name = event.get("name", "world")
    return {
        "greeting": f"hello, {name}",
        "colorama_version": colorama.__version__,
    }


if __name__ == "__main__":
    raw = sys.stdin.read().strip()
    if not raw:
        print(json.dumps({"error": "missing input"}))
        sys.exit(1)
    event = json.loads(raw)
    print(json.dumps(handler(event)))
