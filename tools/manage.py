#!/usr/bin/python
"""Provides configuration management/handling for managing freeradius."""
import argparse
import os
import shutil
import hashlib
import json
import subprocess
import wrapper
import random
import string
import filecmp
import pwd
import urllib.parse
import urllib.request
import datetime
import ssl

REPORTING_HOUR = 4

# user setup
CHARS = string.ascii_uppercase + string.ascii_lowercase + string.digits

# arguments
CHECK = "check"
ADD_USER = "useradd"
BUILD = "build"
GEN_PSWD = "password"

# file handling
FILE_NAME = wrapper.CONFIG_NAME
PREV_FILE = FILE_NAME + ".prev"
AUDIT = "audit.csv"
USER_FOLDER = "users/"
PYTHON_MODS = "mods-config/python"

# env vars
FREERADIUS_REPO = "FREERADIUS_REPO"
NETCONFIG = "NETCONF"
LOG_FILES = "LOG_FILES"
WORK_DIR = "WORKING_DIR"
FLAG_MGMT_LEASE = "LEASE_MGMT"
RPT_HOST = "RPT_HOST"
RPT_TOKEN = "RPT_TOKEN"
RPT_LOCAL = "RPT_LOCAL"


class Env(object):
    """Environment definition."""

    def __init__(self):
        """Init the instance."""
        self.freeradius_repo = None
        self.backing = {}
        self.net_config = None
        self.log_files = None
        self.working_dir = None
        self.mgmt_ips = None
        self.rpt_host = None
        self.rpt_token = None
        self.rpt_local = None

    def add(self, key, value):
        """Add a key, sets into environment."""
        os.environ[key] = value
        if key == FREERADIUS_REPO:
            self.freeradius_repo = value
        elif key == NETCONFIG:
            self.net_config = value
        elif key == LOG_FILES:
            self.log_files = value
        elif key == WORK_DIR:
            self.working_dir = value
        elif key == FLAG_MGMT_LEASE:
            self.mgmt_ips = value
        elif key == RPT_HOST:
            self.rpt_host = value
        elif key == RPT_TOKEN:
            self.rpt_token = value
        elif key == RPT_LOCAL:
            self.rpt_local = value

    def _error(self, key):
        """Print an error."""
        print("{} must be set".format(key))

    def _in_error(self, key, value):
        """Indicate on error."""
        if value is None:
            self._error(key)
            return 1
        else:
            return 0

    def validate(self, full=False):
        """Validate the environment setup."""
        errors = 0
        errors += self._in_error(FREERADIUS_REPO, self.freeradius_repo)
        if full:
            errors += self._in_error(NETCONFIG, self.net_config)
            errors += self._in_error(LOG_FILES, self.log_files)
            errors += self._in_error(WORK_DIR, self.working_dir)
            errors += self._in_error(FLAG_MGMT_LEASE, self.mgmt_ips)
            errors += self._in_error(RPT_HOST, self.rpt_host)
            errors += self._in_error(RPT_TOKEN, self.rpt_token)
            errors += self._in_error(RPT_LOCAL, self.rpt_local)
        if errors > 0:
            exit(1)


def _get_vars(env_file):
    """Get the environment setup."""
    result = Env()
    with open(os.path.expandvars(env_file), 'r') as env:
        for line in env.readlines():
            if line.startswith("#"):
                continue
            parts = line.split("=")
            if len(parts) > 1:
                key = parts[0]
                val = "=".join(parts[1:]).strip()
                if val.startswith('"') and val.endswith('"'):
                    val = val[1:len(val) - 1]
                result.add(key, os.path.expandvars(val))
    result.validate()
    return result


def get_not_cruft(users):
    """Not-cruft users."""
    not_cruft = []
    for flag in ["nocruft", "secondary"]:
        attrs = get_user_attr(users, flag)
        for u in attrs:
            if attrs[u] == "1":
                not_cruft.append(u)
    return not_cruft


def get_file_hash(file_name):
    """Get a sha256 hash of a file."""
    with open(file_name, 'rb') as f:
        sha = hashlib.sha256(f.read())
        return sha.hexdigest()


def _get_exclude(name):
    """Define an rsync exclude."""
    return '--exclude={}'.format(name)


def call(cmd, error_text, working_dir=None):
    """Call for subproces/ing."""
    p = subprocess.Popen(cmd, cwd=working_dir)
    p.wait()
    if p.returncode != 0:
        _smirc("radius call failed")
        print("unable to {}".format(error_text))
        exit(1)


def _get_utils(env):
    """Get utils location."""
    return os.path.join(env.freeradius_repo, PYTHON_MODS, "utils")


def compose(env):
    """Compose the configuration."""
    offset = _get_utils(env)
    rsync = ["rsync",
             "-aczv",
             USER_FOLDER,
             os.path.join(offset, USER_FOLDER),
             "--delete-after",
             _get_exclude("*.pyc"),
             _get_exclude("README.md"),
             _get_exclude("__init__.py"),
             _get_exclude("__config__.py")]
    call(rsync, "rsync user definitions")
    here = os.getcwd()
    composition = ["python",
                   "config_compose.py",
                   "--output",
                   os.path.join(here, FILE_NAME),
                   "--audit",
                   os.path.join(here, AUDIT)]
    call(composition, "compose configuration", working_dir=offset)


def gen_pass(dump, key):
    """Generate password for a user account."""
    if key is None:
        print("no key available")
        exit(1)
    rands = ''.join(random.choice(CHARS) for _ in range(64))
    encoded = wrapper.encrypt(rands, key)
    raw = wrapper.decrypt(encoded, key)
    if rands != raw:
        print("encrypt/decrypt problem")
        exit(1)
    if dump:
        print("password:")
        print(raw)
        print("config file encoded")
        print(encoded)
    else:
        return (raw, encoded)


def add_user(key):
    """Add a new user definition."""
    print("please enter the user name:")
    named = input()
    print("please enter the phabricator name to alias (blank to skip)")
    aliased = input()
    alias = ""
    if aliased is not None and len(aliased) > 0:
        alias = "u_obj.attrs = [common.ALIASED + '{}']".format(aliased)
    passes = gen_pass(False, key)
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
{}
""".format(password, alias)
    with open(os.path.join(USER_FOLDER, "user_" + named + ".py"), 'w') as f:
        f.write(user_definition.strip())
    print("{} was created with a password of {}".format(named, raw))


def get_report_data(env, name):
    """GET or POST report data."""
    report_url = "{}/reports/view/{}?raw=true".format(env.rpt_host, name)
    return make_report_req(env, report_url, None).decode("utf-8")


def make_report_req(env, endpoint, data):
    """Make a report request."""
    ctx = None
    if env.rpt_local == "1":
        ctx = ssl.create_default_context()
        ctx.check_hostname = False
        ctx.verify_mode = ssl.CERT_NONE
    r = urllib.request.urlopen(endpoint, data=data, context=ctx)
    if r.getcode() != 200:
        raise Exception("invalid report request")
    resp = r.read()
    print(resp)
    return resp


def post_content(env, page, content):
    """Post content to a wiki page."""
    report_url = "{}/reports/upload?session={}".format(env.rpt_host,
                                                       env.rpt_token)
    data = {"name": page, "content": content}
    payload = urllib.parse.urlencode(data)
    make_report_req(env, report_url, payload.encode("utf-8"))


def get_user_attr(user, key):
    """Get user attributes."""
    attributes = {}
    for u in user:
        attrs = [x for x in user[u][wrapper.ATTR] if x.startswith(key + "=")]
        if len(attrs) == 1:
            attributes[u] = attrs[0].split("=")[1]
    return attributes


def get_user_resolutions(user):
    """Get user resolutions."""
    return get_user_attr(user, "alias")


def resolve_user(user_name, user_resolutions):
    """Resolve user names."""
    user = user_name
    if user in user_resolutions:
        user = user_resolutions[user]
    else:
        user = user_name.split(".")[1]
    return "[@{}](/p/{})".format(user, user)


def update_assignments(env):
    """Update assignments report."""
    import csv
    lines = [("user", "vlan", "mac"), ("---", "---", "---")]
    with open(AUDIT, 'r') as f:
        r = csv.reader(f)
        for row in r:
            u = row[0]
            v = row[1]
            m = row[2]
            lines.append((u, v, m))
    content = _create_header()
    for l in lines:
        content = content + "| {} | {} | {} |\n".format(l[0], l[1], l[2])
    post_content(env, "assigned", content)


def update_membership(env, running_config):
    """Update wiki pages with config information for VLANs."""
    defs = {}
    with open(running_config, 'r') as f:
        defs = json.loads(f.read())
    users = defs[wrapper.USERS]
    vlans = {}
    for user in sorted(users.keys()):
        vlan_parts = user.split(".")
        vlan = vlan_parts[0].upper()
        if vlan not in vlans:
            vlans[vlan] = []
        vlans[vlan].append(user)
    first = True
    outputs = [("vlan", "user"), ("---", "---")]
    user_resolved = get_user_resolutions(users)
    tracked = []
    for vlan in sorted(vlans.keys()):
        if not first:
            outputs.append(("-", "-"))
        first = False
        for user in vlans[vlan]:
            user_name = resolve_user(user, user_resolved)
            tracking = "{}.{}".format(vlan, user_name)
            if tracking not in tracked:
                outputs.append((vlan, user_name))
                tracked.append(tracking)
    content = _create_header()
    for output in outputs:
        content = content + "| {} | {} |\n".format(output[0], output[1])
    post_content(env, "membership", content)


def update_leases(env, conf):
    """Update the wiki with lease information."""
    leases = {}
    lease_unknown = []
    statics = []
    mgmts = []
    try:
        raw = get_report_data(env, "dns")
        for line in raw.split("\n"):
            if len(line.strip()) == 0:
                continue
            try:
                parts = line.split(" ")
                time = parts[0]
                mac = wrapper.convert_mac(parts[1])
                ip = parts[2]
                init = [ip]
                is_static = "dynamic"
                if time == "static":
                    is_static = "static"
                    statics.append(mac)
                else:
                    lease_unknown.append(mac)
                if ip.startswith(env.mgmt_ips):
                    mgmts.append(mac)
                if mac not in leases:
                    leases[mac] = []
                leases[mac].append("{} ({})".format(ip, is_static))
            except Exception as e:
                print("error parsing line: " + line)
                print(str(e))
                continue
    except Exception as e:
        print("error parsing leases.")
        print(str(e))
    user_resolutions = get_user_resolutions(conf)
    for user in conf:
        user_name = resolve_user(user, user_resolutions)
        macs = conf[user][wrapper.MACS]
        port = conf[user][wrapper.PORT]
        auto = conf[user][wrapper.WILDCARD]
        for lease in leases:
            port_by = lease in port
            leased = lease in macs
            is_wildcard = False
            for wild in auto:
                if is_wildcard:
                    break
                for l in leases[lease]:
                    if wild in l:
                        is_wildcard = True
                        break
            if leased or port_by or is_wildcard:
                leases[lease].append(user_name)
                if lease in lease_unknown:
                    while lease in lease_unknown:
                        lease_unknown.remove(lease)
            if port_by:
                leases[lease].append("port-bypass")
            if is_wildcard and not port_by:
                leases[lease].append("auto-assigned")

    def is_mgmt(lease):
        """Check if a management ip."""
        return lease in mgmts

    def is_normal(lease):
        """Check if a 'normal' ip."""
        return not is_mgmt(lease)
    content = _create_header()
    content = content + _create_lease_table(env,
                                            leases,
                                            lease_unknown,
                                            statics,
                                            "normal",
                                            is_normal)
    content = content + _create_lease_table(env,
                                            leases,
                                            lease_unknown,
                                            statics,
                                            "management",
                                            is_mgmt)
    post_content(env, "leases", content)


def _create_lease_table(env, leases, unknowns, statics, header, filter_fxn):
    """Create a lease wiki table output."""
    outputs = []
    outputs.append(["mac", "attributes"])
    outputs.append(["---", "---"])
    report_objs = []
    for lease in sorted(leases.keys()):
        if not filter_fxn(lease):
            continue
        current = leases[lease]
        attrs = []
        lease_value = lease
        for obj in sorted(set(current)):
            attrs.append(obj)
        if lease in unknowns and lease not in statics:
            report_objs.append(lease_value)
            lease_value = "**{}**".format(lease_value)
        cur_out = [lease_value]
        cur_out.append(" ".join(attrs))
        outputs.append(cur_out)
    content = "\n\n# " + header + "\n\n----\n\n"
    for output in outputs:
        content = content + "| {} | {} |\n".format(output[0],
                                                   output[1])
    if len(report_objs) > 0:
        _smirc("unknown leases (" + header + "): " + ", ".join(report_objs))
    return content


def _smirc(text):
    """Sending via smirc."""
    try:
        import smirc
    except ImportError as e:
        print(text)
        return
    print("smirc: {}".format(text))
    try:
        smirc.run(arguments=[text])
    except smirc.SMIRCError as e:
        print("smirc error")
        print(str(e))


def _get_date_offset(days):
    """Create a date-offset with formatting."""
    return (datetime.date.today() -
            datetime.timedelta(days)).strftime("%Y-%m-%d")


def delete_if_exists(file_name):
    """Delete a file if it exists."""
    if os.path.exists(file_name):
        os.remove(file_name)


def _create_header():
    """Create a report header."""
    return ""


def daily_report(env, running_config):
    """Write daily reports."""
    today = datetime.datetime.now()
    hour = today.hour
    report_indicator = env.working_dir + "indicator"
    if hour != REPORTING_HOUR:
        delete_if_exists(report_indicator)
        return
    if os.path.exists(report_indicator):
        return
    print('SIGHUP radius')
    radius_pid = "/var/run/radiusd/radiusd.pid"
    pid = None
    if os.path.exists(radius_pid):
        with open(radius_pid) as f:
            pid = "".join(f.readlines()).strip()
    if pid is not None:
        call(['kill', '-9', pid], 'sighup radius')
    print('completing daily reports')
    with open(report_indicator, 'w') as f:
        f.write("")
    output = env.working_dir + "auths.md"
    call(["python",
          "report_auths.py",
          "--output",
          output],
         "report authorizations",
         working_dir=_get_utils(env))
    auths = None
    optimized = {}
    conf = None
    with open(running_config, 'r') as f:
        conf = json.loads(f.read())[wrapper.USERS]
    not_cruft = get_not_cruft(conf)
    _is_na = "n/a"
    with open(output) as f:
        lines = []
        skip = 0
        for l in f:
            new_line = l
            if skip >= 2:
                parts = l.split("|")
                user = parts[1].strip()
                res = parts[3]
                if user not in optimized:
                    optimized[user] = False
                if _is_na in res:
                    if user not in not_cruft:
                        adj = []
                        for x in parts:
                            if len(x.strip()) == 0:
                                adj.append(x)
                            else:
                                cruft_mark = x.replace(_is_na,
                                                       _is_na + " (cruft)")
                                adj.append("**{}**".format(cruft_mark))
                        new_line = "|".join(adj)
                else:
                    optimized[user] = True
            lines.append(new_line)
            skip += 1
        auths = "".join(lines)
    post_content(env, "auths", _create_header() + auths)
    update_leases(env, conf)
    suggestions = []
    for u in optimized:
        if not optimized[u]:
            if u not in not_cruft:
                suggestions.append("drop user {}".format(u))
    if len(suggestions) > 0:
        _smirc("\n".join(sorted(suggestions)))


def build():
    """Build and apply a user configuration."""
    env = _get_vars("/etc/environment")
    env.validate(full=True)
    os.chdir(env.net_config)
    compose(env)
    new_config = os.path.join(env.net_config, FILE_NAME)
    run_config = os.path.join(env.freeradius_repo, PYTHON_MODS, FILE_NAME)
    diff = filecmp.cmp(new_config, run_config)
    if not diff:
        print('change detected')
        shutil.copyfile(run_config, run_config + ".prev")
        shutil.copyfile(new_config, run_config)
        u = pwd.getpwnam("radiusd")
        os.chown(run_config, u.pw_uid, u.pw_gid)
        update_membership(env, run_config)
        update_assignments(env)
        hashed = get_file_hash(FILE_NAME)
        git = "latest commit"
        git_indicator = env.working_dir + "git"
        if os.path.exists(git_indicator):
            with open(git_indicator, 'r') as f:
                git = f.read().strip()
        status = "ready"
        _smirc("{} -> {} ({})".format(status, git, hashed))
    daily_report(env, run_config)


def check():
    """Check composition."""
    env = _get_vars("$HOME/.config/epiphyte/env")
    if os.path.exists(FILE_NAME):
        shutil.copyfile(FILE_NAME, PREV_FILE)
    compose(env)
    if os.path.exists(FILE_NAME):
        print(get_file_hash(FILE_NAME))
        output = None
        with open(FILE_NAME, 'r') as f:
            j = json.loads(f.read())
            output = json.dumps(j,
                                sort_keys=True,
                                indent=4,
                                separators=(',', ': '))
        with open(FILE_NAME, 'w') as f:
            f.write(output)


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser()
    parser.add_argument('action',
                        nargs='?',
                        choices=[CHECK, ADD_USER, BUILD, GEN_PSWD],
                        default=CHECK)
    parser.add_argument('--key', type=str)
    args = parser.parse_args()
    key = None
    if args.key:
        key = wrapper.convert_key(args.key)
    if args.action == CHECK:
        check()
    elif args.action == BUILD:
        try:
            build()
        except Exception as e:
            _smirc("build error")
            print(str(e))
    elif args.action == ADD_USER:
        add_user(key)
    elif args.action == GEN_PSWD:
        gen_pass(True, key)


if __name__ == "__main__":
    main()
