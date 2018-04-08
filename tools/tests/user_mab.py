"""User with admin and dev and various macs."""
import users.__config__ as __config__
import users.common as common
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.bypass = ["112233445567"]
normal.vlan = "dev"
normal.password = 'ac0ae0d888d0e71c3dae227377a8601e'
normal.limited = ['abcdef123456']
normal.mab_only = True
