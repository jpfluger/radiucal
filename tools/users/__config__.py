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
        self.vlan = None
        self.expires = None
        self.disabled = False
        self.inherits = None
        self.owns = []
        self._bypass = None
        self.mab_only = False

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

    def mab(self, mac, vlan=None):
        """Set a MAC as MAB."""
        v = vlan
        if vlan is None:
            v = self.vlan
        if v is None:
            raise Exception("mab before vlan assigned")
        if self._bypass is None:
            self._bypass = {}
        self._bypass[mac] = v

    def bypassed(self):
        """Get MAB bypassed MACs."""
        if self._bypass is None:
            return []
        return list(self._bypass.keys())

    def bypass_vlan(self, mac):
        """Get a MAB bypassed VLAN."""
        return self._bypass[mac]

    def _check_macs(self, against, previous=[]):
        """Check macs."""
        if against is not None and len(against) > 0:
            already_set = self.macs + previous
            if self._bypass is not None:
                already_set = already_set + list(self._bypass.keys())
            if previous is not None:
                already_set = already_set + previous
            for mac in against:
                if not is_mac(mac):
                    return mac
                if mac in already_set:
                    return mac
        return None

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
        has_mac = False
        knowns = []
        if self._bypass is not None:
            knowns = self._bypass.keys()
        for mac_group in [self.macs, knowns]:
            if mac_group is not None and len(mac_group) > 0:
                has_mac = True
        if not has_mac:
            return self.report("no macs listed")
        for mac in self.macs:
            if not is_mac(mac):
                return False
        if self.password is None or len(self.password) == 0:
            return self.report("no or short password")
        if len(knowns) > 0:
            for mac in knowns:
                if not is_mac(mac, category='bypass'):
                    return False
        for c in [self._check_macs(self.owns)]:
            if c is not None:
                return self.report("invalid mac (known): {}".format(c))
        if len(self.macs) != len(set(self.macs)):
            return self.report("macs not unique")
        return True
