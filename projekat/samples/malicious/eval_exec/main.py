def handler(event=None):
    code = event.get("code", "1+1") if event else "1+1"
    return {"result": eval(code)}
