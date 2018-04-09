"""User with admin and dev and various macs."""
import users.__config__ as __config__
import users.common as common
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.vlan = "dev"
normal.mab("112233445567")
normal.password = 'ac0ae0d888d0e71c3dae227377a8601e'
normal.mab('abcdef123456', vlan=4000)
normal.mab_only = True
