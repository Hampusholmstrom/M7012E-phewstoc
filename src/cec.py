from subprocess import PIPE, run


def _cec_send(device, command):
    proc = run(["cec-client", "-s", "-d", str(device)], stdout=PIPE,
               input=command, encoding="ascii")

    if proc.returncode != 0:
        raise Exception("unexpected return code: %s, output: %s" % (proc.returncode, proc.stdout))


def off(device):
    _cec_send(device, "standby 0")


def on(device):
    _cec_send(device, "on 0")
