#!/usr/bin/env python3

from subprocess import PIPE, run
from time import sleep


def _cec_send(device, command):
    proc = run(["cec-client", "-s", "-d", str(device)], stdout=PIPE,
               input=command, encoding="ascii")

    if proc.returncode != 0:
        raise Exception("unexpected return code: %s, output: %s" % (proc.returncode, proc.stdout))


def cec_off(device):
    _cec_send(device, "standby 0")


def cec_on(device):
    _cec_send(device, "on 0")


# Turn TV on and off twice.
cec_on(1)
sleep(5)
cec_off(1)
sleep(5)

cec_on(1)
sleep(5)
cec_off(1)
sleep(5)
