#!/usr/bin/python
"""Provides configuration management/handling for managing freeradius."""
import argparse
import os
import shutil
import hashlib
import json
import subprocess
import random
import string
import filecmp
import pwd
import urllib.parse
import urllib.request
import datetime
import ssl
import hashlib

# user setup
CHARS = string.ascii_uppercase + string.ascii_lowercase + string.digits
ADDUSER = "adduser"
GENPSWD = "pwd"


def gen_pass(dump):
    """Generate password for a user account."""
    rands = ''.join(random.choice(CHARS) for _ in range(64))
    encoded = hashlib.new('md4', rands.encode("utf_16_le")).digest().hex()
    if dump:
        print("")
        print("password:")
        print(rands)
        print("")
        print("md4:")
        print(encoded)
        print("")
    return (rands, encoded)


def add_user():
    """Add a new user definition."""
    print("please enter the user name:")
    named = input()
    passes = gen_pass(False)
    raw = passes[0]
    password = passes[1]
    user_definition = """
import users.__config__ as __config__
import users.common as common

u_obj = __config__.Assignment()
u_obj.password = '{}'
u_obj.vlan = None
u_obj.group = None
u_obj.macs = None
""".format(password)
    with open(os.path.join("users/", "user_" + named + ".py"), 'w') as f:
        f.write(user_definition.strip())
    print("{} was created with a password of {}".format(named, raw))


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser()
    parser.add_argument('action',
                        nargs='?',
                        choices=[ADDUSER, GENPSWD],
                        default=ADDUSER)
    args = parser.parse_args()
    if args.action == ADDUSER:
        add_user()
    else:
        gen_pass(True)


if __name__ == "__main__":
    main()
