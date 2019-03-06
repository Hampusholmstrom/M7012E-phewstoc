import requests

r = requests.get("https://phewstoc.sladic.se/issleeping/")
if r.status_code:
    print("Asleep!")
else:
    print("Awake!")
