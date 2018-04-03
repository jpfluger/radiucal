#!/usr/bin/python
"""configuration base."""
from datetime import datetime
import re

PASSWORD_LENGTH = 32


def is_mac(wrapper, value, category=None):
    """validate if something appears to be a mac."""
    valid = wrapper.is_mac(value) 
    if not valid:
        cat = ''
        if category is not None:
            cat = " (" + category + ")"
        print('invalid mac detected{} {}'.format(cat, value))
    return valid


class VLAN(object):
    """VLAN definition."""

    def __init__(self, name, number):
        """init the instance."""
        self.name = name
        self.num = number
        self.initiate = None
        self.route = None
        self.net = None
        self.owner = None
        self.desc = None
        self.group = None

    def check(self, wrapper):
        """Check the definition."""
        if self.name is None or len(self.name) == 0 \
           or not isinstance(self.num, int):
            return False
        return True


class Assignment(object):
    """assignment object."""
    def __init__(self):
        self.macs = []
        self.password = ""
        self.bypass = []
        self.vlan = None
        self.disable = {}
        self.no_login = False
        self.attrs = None
        self.expires = None
        self.disabled = False
        self.inherits = None
        self.port_bypass = []
        self.wildcard = []
        self.group = None

    def _compare_date(self, value, regex, today):
        """compare date."""
        matches = regex.findall(value)
        for match in matches:
            as_date = datetime.strptime(match, '%Y-%m-%d')
            return as_date < today
        return None

    def report(self, cause):
        """report an issue."""
        print(cause)
        return False

    def copy(self, other):
        """copy/inherit from another entity."""
        self.password = other.password
        self.attrs = other.attrs
        self.macs = set(self.macs + other.macs)
        self.group = other.group

    def check(self, wrapper):
        """check the assignment definition."""
        if self.inherits:
            self.copy(self.inherits)
        today = datetime.now()
        today = datetime(today.year, today.month, today.day)
        regex = re.compile(r'\d{4}[-/]\d{2}[-/]\d{2}')
        if self.expires is not None:
            res = self._compare_date(self.expires, regex, today)
            if res is not None:
                self.disabled = res
            else:
                return self.report("invalid expiration")
        if self.vlan is None or len(self.vlan) == 0:
            return self.report("no vlan assigned")
        if self.macs is None or len(self.macs) == 0:
            return self.report("no macs listed")
        for mac in self.macs:
            if not is_mac(wrapper, mac):
                return False
        if self.password is None or len(self.password) < 32:
            return self.report("no or short password")
        for c in self.password:
            try:
                int(c)
            except ValueError:
                if c == '|' or c == '.':
                    pass
                else:
                    return self.report("invalid character in password")

        if self.bypass is not None and len(self.bypass) > 0:
            for mac in self.bypass:
                if not is_mac(wrapper, mac, category='bypass'):
                    return False 
        if self.port_bypass is not None and len(self.port_bypass):
            already_set = self.macs
            if self.bypass is not None:
                already_set = already_set + self.bypass
            for mac in self.port_bypass:
                if not is_mac(wrapper, mac):
                    return False
                if mac in already_set:
                    return self.report("invalid port bypass mac")
        if len(self.macs) != len(set(self.macs)):
            return self.report("macs not unique")
        if self.disable is not None and len(self.disable) > 0:
            if isinstance(self.disable, dict):
                for key in self.disable.keys():
                    val = self.disable[key]
                    res = self._compare_date(val, regex, today)
                    if res is not None:
                        if res:
                            print("{0} has been time-disabled".format(key))
                            if key in self.bypass:
                                self.bypass.remove(key)
                            if key in self.macs:
                                self.macs.remove(key)
                    else:
                        return self.report("invalid MAC date")
        if self.attrs and len(self.attrs) > 0:
            uniq_attr = []
            for attr in self.attrs:
                parts = attr.split("=")
                if len(parts) != 2:
                    return self.report("attributes must be: key=value")
                uniq_attr.append(parts[0])
            if len(uniq_attr) != len(set(uniq_attr)):
                return self.report("attribute keys must be unique")
        if self.group is None:
            return self.report("no group specified")
        return True
