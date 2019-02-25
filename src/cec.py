from subprocess import PIPE, run


class CECError(Exception):
    """
    An error code was returned by the CEC command.
    """
    pass


def _cec_send(device, command):
    proc = run(["cec-client", "-s", "-d", str(device)], stdout=PIPE,
               input=command, encoding="ascii")

    if proc.returncode != 0:
        raise CECError("unexpected return code: %s, output: %s" % (proc.returncode, proc.stdout))


def off(device):
    _cec_send(device, "standby 0")


def on(device):
    _cec_send(device, "on 0")
