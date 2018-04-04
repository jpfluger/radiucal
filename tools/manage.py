#!/usr/bin/python
"""Provides configuration management/handling for managing freeradius."""
FLAG_MGMT_LEASE = "LEASE_MGMT"


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
