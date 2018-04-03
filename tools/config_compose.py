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


class ConfigMeta(object):
    """configuration meta information."""

    def __init__(self):
        """init the instance."""
        self.passwords = []
        self.macs = []
        self.bypasses = []
        self.vlans = []
        self.all_vlans = []
        self.user_name = []
        self.vlan_users = []
        self.attrs = []
        self.vlan_initiate = []

    def password(self, password):
        """password group validation(s)."""
        if password in self.passwords:
            print("password duplicated")
            exit(-1)
        self.passwords.append(password)

    def bypassed(self, macs):
        """bypass management."""
        for mac in macs:
            if mac in self.bypasses:
                print("already bypassed")
                exit(-1)
            self.bypasses.append(mac)

    def user_macs(self, macs):
        """user+mac combos."""
        self.macs = self.macs + macs
        self.macs = list(set(self.macs))

    def attributes(self, attrs):
        """set attributes."""
        self.attrs = self.attrs + attrs
        self.attrs = list(set(self.attrs))

    def verify(self):
        """verify meta data."""
        for mac in self.macs:
            if mac in self.bypasses:
                print("mac is globally bypassed: " + mac)
                exit(-1)
        for mac in self.bypasses:
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


class EAPUser(object):
    """EAP user definition."""

    def __init__(self, macs, password, attrs, port_bypassed, wildcards):
        """Init the instance."""
        self.macs = macs
        self.password = password
        self.attrs = attrs
        self.port_bypass = port_bypassed
        self.wildcard = wildcards


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


def _process(output, audit):
    """process the composition of users."""
    common_mod = None
    try:
        common_mod = _get_mod("common")
        print("loaded common definitions...")
    except Exception as e:
        print("defaults only...")
    user_objs = {}
    vlans = None
    bypass_objs = {}
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
    vlans_with_users = {}
    user_macs = {}
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
            vlans_with_users[vlan] = vlans[vlan]
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
            port_bypassed = sorted(obj.port_bypass)
            wildcards = sorted(obj.wildcard)
            attrs = []
            if obj.attrs:
                attrs = sorted(obj.attrs)
                meta.attributes(attrs)
            # meta checks
            meta.user_macs(macs)
            if not obj.inherits:
                meta.password(password)
            meta.bypassed(bypass)
            if fqdn in user_objs:
                raise Exception(fqdn + " previously defined")
            # use config definitions here
            if not obj.no_login:
                user_objs[fqdn] = EAPUser(macs,
                                          password,
                                          attrs,
                                          port_bypassed,
                                          wildcards)
            if bypass is not None and len(bypass) > 0:
                for mac_bypass in bypass:
                    if mac_bypass in bypass_objs:

                        raise Exception(mac_bypass + " previously defined")
                    bypass_objs[mac_bypass] = vlan
            user_all = []
            for l in [obj.macs, obj.port_bypass, obj.bypass]:
                user_all += list(l)
            if key not in user_macs:
                user_macs[key] = []
            user_macs[key].append((vlan, sorted(set(user_all))))
    meta.verify()
    with open(output, 'w') as f:
        csv_writer = csv.writer(f, lineterminator=os.linesep)
        for u in user_objs:
            o = user_objs[u]
            for g in o.build():
                csv_writer.writerow(g)
    with open(audit, 'w') as f:
        csv_writer = csv.writer(f, lineterminator=os.linesep)
        for u in user_macs:
            for obj in user_macs[u]:
                vlan = obj[0]
                macs = obj[1]
                for m in macs:
                    csv_writer.writerow([u, vlan, m])


def main():
    """main entry."""
    success = False
    try:
        parser = argparse.ArgumentParser()
        parser.add_argument("--output", type=str, required=True)
        parser.add_argument("--audit", type=str, required=True)
        args = parser.parse_args()
        _process(args.output, args.audit)
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
