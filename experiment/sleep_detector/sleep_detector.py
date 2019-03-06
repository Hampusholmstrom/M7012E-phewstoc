import requests

r = requests.get("https://phewstoc.sladic.se/issleeping/")
if r.status_code == 418:
    print("Asleep!")
else:
    print("Awake!")
