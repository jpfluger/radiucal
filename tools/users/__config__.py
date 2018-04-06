#!/usr/bin/python
"""configuration base."""
from datetime import datetime
import re


def is_valid_mac(possible_mac):
    """check if an object is a mac."""
    valid = False
    if len(possible_mac) == 12:
        valid = True
        for c in possible_mac:
            if c not in ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
                         'a', 'b', 'c', 'd', 'e', 'f']:
                valid = False
                break
    return valid


def is_mac(value, category=None):
    """validate if something appears to be a mac."""
    valid = is_valid_mac(value)
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

    def check(self, ):
        """Check the definition."""
        if self.name is None or len(self.name) == 0 \
           or not isinstance(self.num, int):
            return False
        return True


class Assignment(object):
    """assignment object."""

    def __init__(self):
        """Init the instance."""
        self.macs = []
        self.password = ""
        self.bypass = []
        self.vlan = None
        self.disable = {}
        self.no_login = False
        self.expires = None
        self.disabled = False
        self.inherits = None
        self.owns = []
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
        self.macs = set(self.macs + other.macs)
        self.group = other.group

    def check(self):
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
            if not is_mac(mac):
                return False
        if self.password is None or len(self.password) == 0:
            return self.report("no or short password")
        if self.bypass is not None and len(self.bypass) > 0:
            for mac in self.bypass:
                if not is_mac(mac, category='bypass'):
                    return False
        if self.owns is not None and len(self.owns):
            already_set = self.macs
            if self.bypass is not None:
                already_set = already_set + self.bypass
            for mac in self.owns:
                if not is_mac(mac):
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
        if self.group is None:
            return self.report("no group specified")
        return True
