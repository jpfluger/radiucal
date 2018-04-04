"""User with a disabled setting."""
import users.__config__ as __config__
import users.common as common
disabled = "11ff33445566"
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.bypass = [disabled]
normal.disable = {disabled: "2017-01-01"}
normal.vlan = "dev"
normal.group = 'test'
normal.password = '180599206f5dffdd3881a121110da8ad'
