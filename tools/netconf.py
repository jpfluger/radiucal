#!/usr/bin/python
"""composes the config from user definitions."""
import argparse
import os
import users
import users.__config__
import importlib
import csv

# file indicators
IND_DELIM = "_"
USER_INDICATOR = "user" + IND_DELIM
VLAN_INDICATOR = "vlan" + IND_DELIM
VLAN_UNAUTH = "4000"
AUTH_PHASE_ONE = "PEAP"
AUTH_PHASE_TWO = "MSCHAPV2"


class ConfigMeta(object):
    """configuration meta information."""

    def __init__(self):
        """init the instance."""
        self.passwords = []
        self.macs = []
        self.vlans = []
        self.all_vlans = []
        self.user_name = []
        self.vlan_users = []
        self.vlan_initiate = []
        self.extras = []

    def password(self, password):
        """password group validation(s)."""
        if password in self.passwords:
            print("password duplicated")
            exit(-1)
        self.passwords.append(password)

    def extra(self, macs):
        """Limited macs."""
        for mac in macs:
            if mac in self.extras:
                print("mac already known as extra: " + mac)
                exit(-1)
            self.extras.append(mac)

    def user_macs(self, macs):
        """user+mac combos."""
        self.macs = self.macs + macs
        self.macs = list(set(self.macs))

    def verify(self):
        """verify meta data."""
        for mac in self.macs:
            if mac in self.extras:
                print("mac is flagged extra: " + mac)
                exit(-1)
        for mac in self.extras:
            if mac in self.macs:
                print("mac is user assigned: " + mac)
                exit(-1)
        used_vlans = set(self.vlans + self.vlan_initiate)
        if len(used_vlans) != len(set(self.all_vlans)):
            print("unused vlans detected")
            exit(-1)
        for ref in used_vlans:
            if ref not in self.all_vlans:
                print("reference to unknown vlan: " + ref)
                exit(-1)

    def vlan_user(self, vlan, user):
        """indicate a vlan was used."""
        self.vlans.append(vlan)
        self.vlan_users.append(vlan + "." + user)
        self.user_name.append(user)

    def vlan_to_vlan(self, vlan_to):
        """VLAN to VLAN mappings."""
        self.vlan_initiate.append(vlan_to)


def _get_mod(name):
    """import the module dynamically."""
    return importlib.import_module("users." + name)


def _load_objs(name, typed):
    mod = _get_mod(name)
    for key in dir(mod):
        obj = getattr(mod, key)
        if not isinstance(obj, typed):
            continue
        yield obj


def _get_by_indicator(indicator):
    """get by a file type indicator."""
    return [x for x in sorted(users.__all__) if x.startswith(indicator)]


def _common_call(common, method, entity):
    """make a common mod call."""
    obj = entity
    if common is not None and method in dir(common):
        call = getattr(common, method)
        if call is not None:
            obj = call(obj)
    return obj


def check_object(obj):
    """Check an object."""
    return obj.check()


def _process(output):
    """process the composition of users."""
    common_mod = None
    try:
        common_mod = _get_mod("common")
        print("loaded common definitions...")
    except Exception as e:
        print("defaults only...")
    vlans = None
    meta = ConfigMeta()
    for v_name in _get_by_indicator(VLAN_INDICATOR):
        print("loading vlan..." + v_name)
        for obj in _load_objs(v_name, users.__config__.VLAN):
            if vlans is None:
                vlans = {}
            if not check_object(obj):
                exit(-1)
            num_str = str(obj.num)
            for vk in vlans.keys():
                if num_str == vlans[vk]:
                    print("vlan number defined multiple times...")
                    exit(-1)
            vlans[obj.name] = num_str
            if obj.initiate is not None and len(obj.initiate) > 0:
                for init_to in obj.initiate:
                    meta.vlan_to_vlan(init_to)
    if vlans is None:
        raise Exception("missing required config settings...")
    meta.all_vlans = vlans.keys()
    store = Store()
    for f_name in _get_by_indicator(USER_INDICATOR):
        print("composing..." + f_name)
        for obj in _load_objs(f_name, users.__config__.Assignment):
            obj = _common_call(common_mod, 'ready', obj)
            key = f_name.replace(USER_INDICATOR, "")
            if not key.isalnum():
                print("does not meet naming requirements...")
                exit(-1)
            vlan = obj.vlan
            if vlan not in vlans:
                raise Exception("no vlan defined for " + key)
            store.add_vlan(vlan, vlans[vlan])
            meta.vlan_user(vlan, key)
            fqdn = vlan + "." + key
            if not check_object(obj):
                print("did not pass check...")
                exit(-1)
            if obj.disabled:
                print("account is disabled or has expired...")
                continue
            macs = sorted(obj.macs)
            password = obj.password
            bypass = sorted(obj.bypass)
            owned = sorted(obj.owns)
            limited = sorted(obj.limited)
            # meta checks
            meta.user_macs(macs)
            if not obj.inherits:
                meta.password(password)
            meta.extra(bypass)
            meta.extra(owned)
            meta.extra(limited)
            # use config definitions here
            if not obj.no_login:
                store.add_user(fqdn, macs, password, owned, limited)

            def add_mac(macs, vlan_id):
                """Add a mac+vlan to the store."""
                if macs and len(macs) > 0:
                    for m in macs:
                        store.add_mab(m, vlan_id)
            add_mac(bypass, vlan)
            add_mac(limited, VLAN_UNAUTH)
            user_all = []
            for l in [obj.macs, obj.owns, obj.bypass, obj.limited]:
                user_all += list(l)
            store.add_audit(fqdn, sorted(set(user_all)))
    meta.verify()
    # audit outputs
    with open(output + "audit.csv", 'w') as f:
        csv_writer = csv.writer(f, lineterminator=os.linesep)
        for a in sorted(store.get_tag(store.audit)):
            p = a[0].split(".")
            for m in a[1]:
                csv_writer.writerow([p[1], p[0], m])
    # eap_users and preauth
    manifest = []
    with open(output + "eap_users", 'w') as f:
        for u in store.get_eap_user():
            f.write('"{}" {}\n\n'.format(u[0], AUTH_PHASE_ONE))
            f.write('"{}" {} hash:{} [2]\n'.format(u[0], AUTH_PHASE_TWO, u[1]))
            write_vlan(f, u[2])
        for u in store.get_eap_mab():
            up = u[0].upper()
            f.write('"{}" MD5 "{}"\n'.format(up, up))
            if u[2]:
                write_vlan(f, u[1])
            else:
                f.write("\n")
            manifest.append((u[0], u[0]))
    for u in store.get_tag(store.umac):
        manifest.append((u[0], u[1]))
    with open(output + "manifest", 'w') as f:
        for m in sorted(manifest):
            f.write("{}.{}\n".format(m[0], m[1]).lower())


def write_vlan(f, vlan_id):
    """Write vlan assignment for login."""
    f.write('radius_accept_attr=64:d:13\n')
    f.write('radius_accept_attr=65:d:6\n')
    f.write('radius_accept_attr=81:s:{}\n\n'.format(vlan_id))


class Store(object):
    """Storage object."""

    def __init__(self):
        """Init the instance."""
        self._data = []
        self.vlan = "VLAN"
        self.umac = "UMAC"
        self.pwd = "PWD"
        self.owned = "OWNS"
        self.mac = "MAC"
        self.audit = "AUDIT"
        self._users = []
        self._macs = []
        self._vlans = {}
        self._vlans[VLAN_UNAUTH] = VLAN_UNAUTH

    def get_tag(self, tag):
        """Get tagged items."""
        for item in self._data:
            if item[0] == tag:
                yield item[1:]

    def add_vlan(self, vlan_name, vlan_id):
        """Add a vlan item."""
        self._vlans[vlan_name] = vlan_id
        self._add(self.vlan, vlan_name, vlan_id)

    def _add(self, tag, key, value):
        """Backing tagged add."""
        self._data.append([tag, key, value])

    def add_user(self,
                 username,
                 macs,
                 password,
                 owns,
                 limited):
        """Add a user definition."""
        if username in self._users:
            raise Exception("{} already defined".format(username))
        self._users.append(username)
        for m in macs:
            self._add(self.umac, username, m)
        self._add(self.pwd, username, password)
        for p in owns:
            self._add(self.owned, username, p)

    def add_mab(self, mac, vlan):
        """Add a MAB."""
        if mac in self._macs:
            raise Exception("{} already defined".format(mac))
        self._macs.append(mac)
        self._add(self.mac, mac, vlan)

    def add_audit(self, user, objs):
        """Add an audit entry."""
        self._add(self.audit, user, objs)

    def get_eap_mab(self):
        """Get eap entries for MAB."""
        for m in self.get_tag(self.mac):
            v = self._get_vlan(m[1])
            yield [m[0], v, v != VLAN_UNAUTH]

    def get_eap_user(self):
        """Get eap users."""
        for u in self.get_tag(self.pwd):
            vlan = u[0].split(".")[0]
            yield [u[0], u[1], self._get_vlan(vlan)]

    def _get_vlan(self, name):
        """Get vlans."""
        return self._vlans[name]


def main():
    """main entry."""
    success = False
    try:
        parser = argparse.ArgumentParser()
        parser.add_argument("--output", type=str, required=True)
        args = parser.parse_args()
        _process(args.output)
        success = True
    except Exception as e:
        print('unable to compose')
        print(str(e))
    if success:
        print("success")
        exit(0)
    else:
        print("failure")
        exit(1)


if __name__ == "__main__":
    main()
