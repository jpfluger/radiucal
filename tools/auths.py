#!/usr/bin/python
"""Report authorization information."""
import datetime as dt
import argparse
import json
import wrapper
import os

_KEY = "->"
_NA = "n/a"
_DENY = "denied"


def _new_key(user, mac):
    """Create a key."""
    return "{}{}{}".format(user, _KEY, mac)


def _file(day_offset, auth_info, logs):
    """Read a file."""
    uuid_log = {}
    file_name = os.path.join(logs, "trace.log.{}".format(day_offset))
    if not os.path.exists(file_name):
        print("{} does not exist".format(file_name))
        return
    with open(file_name, 'r') as f:
        for l in f:
            parts = l.split("->")
            uuid = parts[0].split(":")[3].strip()
            data = parts[1]
            is_accept = "Tunnel-Type" in data
            is_response = "('Response', 2)" in data
            if is_accept or is_response:
                if uuid in uuid_log:
                    user = uuid_log[uuid]
                    auth_cur = auth_info[user]
                    if auth_cur != day_offset:
                        auth_info[user] = day_offset
                        if is_accept:
                            auth_info[user] += "?"
            else:
                if "User-Name" in data:
                    idx = data.index("User-Name") + 13
                    user_start = data[idx:]
                    user_start = user_start[:user_start.index(")") - 1]
                    calling = None
                    if "Calling-Station-Id" in data:
                        calling_station = data.index("Calling-Station-Id") + 22
                        calling = data[calling_station:]
                        calling = calling[:calling.index(")") - 1]
                        calling = calling.replace(":",
                                                  "").replace("-",
                                                              "").lower()
                        key = _new_key(user_start, calling)
                        uuid_log[uuid] = key
                        if key not in auth_info:
                            auth_info[key] = "{} ({})".format(_DENY,
                                                              day_offset)


def main():
    """Accept/reject reporting."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--days", type=int, default=10)
    parser.add_argument("--config",
                        type=str,
                        default="/etc/raddb/mods-config/python/network.json")
    parser.add_argument("--output",
                        type=str,
                        default=None)
    parser.add_argument("--logs",
                        type=str,
                        default="/var/log/radius/freepydius")
    args = parser.parse_args()
    config = None
    authd = {}
    with open(args.config) as f:
        j = json.loads(f.read())
        users = j[wrapper.USERS]
        for u in users:
            for m in users[u][wrapper.MACS]:
                k = _new_key(u, m)
                authd[k] = _NA
    today = dt.date.today()
    for x in reversed(range(1, args.days + 1)):
        _file("{}".format(today - dt.timedelta(days=x)), authd, args.logs)
    lines = []
    lines.append("| user | mac | last |")
    lines.append("| ---  | --- | ---  |")
    denied = []
    cruft = []
    rest = []
    for item in sorted(authd.keys()):
        val = authd[item]
        if _NA in val:
            cruft.append(item)
        elif _DENY in val:
            denied.append(item)
        else:
            rest.append(item)
    for item in denied + cruft + rest:
        on = authd[item]
        parts = item.split(_KEY)
        if on is None:
            on = ""
        lines.append("| {} | {} | {} |".format(parts[0], parts[1], on))
    if args.output is None:
        for l in lines:
            print(l)
    else:
        with open(args.output, 'w') as f:
            f.write("\n".join(lines))


if __name__ == "__main__":
    main()
