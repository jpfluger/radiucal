"""pwd user."""

import users.__config__ as __config__
import users.common as common
normal = __config__.Assignment()
normal.macs = [common.VALID_MAC]
normal.vlan = "dev"
normal.password = '9eebc40b1fb48ddb3309d824abc28ae3'

admin = __config__.Assignment()
admin.macs = normal.macs
admin.vlan = "prod"
admin.macs = [common.DROP_MAC]
admin.password = '9eebc40b1fb48ddb3309d824abc28ae3'
