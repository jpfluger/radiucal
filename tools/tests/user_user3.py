"""User with inheritance."""
import users.__config__ as __config__
normal = __config__.Assignment()
normal.macs = ["001122334455"]
normal.vlan = "dev"
normal.attrs = ["test=test"]
normal.port_bypass = ["001122221100"]
normal.wildcard = ["abc"]
normal.group = 'test'

admin = __config__.Assignment()
admin.inherits = normal
admin.vlan = "prod"
admin.group = 'admin'
normal.password = 'e2192da00a1ccba417ec515395a044f7'
