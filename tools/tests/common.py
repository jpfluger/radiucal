"""Common testing definitions."""
VALID_MAC = "001122334455"
DROP_MAC = "ffffffffffff"


def ready(obj):
    """Ready obj."""
    if obj.macs and len(obj.macs) == 1:
        if obj.macs[0] == DROP_MAC:
            obj.disabled = True
    return obj
