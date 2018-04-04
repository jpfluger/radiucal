"""User with admin and dev and various macs."""
import users.__config__ as __config__
import users.common as common
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.bypass = ["112233445566"]
normal.vlan = "dev"
normal.group = 'test'
normal.password = 'ac0ae0d888d0e71c3dae227377a86011'

admin = __config__.Assignment()
admin.macs = normal.macs
admin.vlan = "prod"
admin.group = "adm"
admin.password = 'ac0ae0d888d0e71c3dae227377a86012'
