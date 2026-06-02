def handler(event=None):
    return {"message": "hello from oblak", "event": event}


if __name__ == "__main__":
    print(handler({"from": "cli"}))

