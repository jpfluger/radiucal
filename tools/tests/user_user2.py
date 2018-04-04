"""User in multiple vlans."""
import users.__config__ as __config__
normal = __config__.Assignment()
normal.macs = ["001122334455", "aabbccddeeff"]
normal.vlan = "dev"
normal.group = 'test'
normal.password = 'c2f7aacb0c14b7fdfaddd4679102359c'

admin = __config__.Assignment()
admin.macs = normal.macs
admin.vlan = "prod"
admin.group = 'admin'
admin.password = 'c2f7aacb0c14b7fdfaddd4679102359d'
