radiucal
===

Designed for using a go proxy+hostapd as an 802.1x RADIUS server for network authentication (or how to live without freeradius)

# purpose

This is a go proxy+hostapd setup that provides a very simple configuration to manage 802.1x authentication and management on a LAN.

Expectations:
* Running on archlinux as a host/server
* hostapd can do a lot with EAP and RADIUS as a service, this should serve as an exploration of these features
* Fully replace freeradius for 802.1x/AAA/etc.
* utilize a trimmed version of hostapd (from here on called `hostapd-server`) using a config and PKGBUILD [here](https://github.com/epiphyte/pkgbuilds/tree/master/hostapd)

## AAA

* Authentication (Your driver's license proves that you're the person you say you are)
* Authorization (Your driver's license lets you drive your car, motorcycle, or CDL)
* Accounting (A log shows that you've driven on these roads at a certain date)

## Goals

* Support a port-restricted LAN (+wifi) in a controlled, physical area
* Provide a singular authentication strategy for supported clients using peap+mschapv2 (no CA validation).
* Windows 10
* Linux (any supporting modern versions of NetworkManager or systemd-networkd when server/headless)
* Android 7+
* Map authenticated user+MAC combinations to specific VLANs
* Support MAC-based authentication (bypass) for systems that can not authenticate themselves
* Integrate with a variety of network equipment
* Avoid client-issued certificates (and management)
* Centralized configuration file
* As few open endpoints as possible on the radius server (only open ports 1812 and 1813 for radius)

**These goals began with our usage of freeradius and continue to be vital to our operation**

## Proxy

radiucal is a go proxy that receives UDP packets and routes them along (namely to hostapd/another radius server)

the proxy:
* provides a modularized/plugin approach to handle preauth, auth, and accounting actions
* can support user+mac filtering, logging, trace output, and simple stat output via plugins
* provides a cut-in for more plugins
* overrides the concept of "radius_clients" as all will have to have a single shared secret

# install

install from the epiphyte [repository](https://mirror.epiphyte.network/repos)
```
pacman -S hostapd-server radiucal
```

adminstrative machines should install
```
pacman -S radiucual-utils
```

matching tooling for utils on a server
```
pacman -S radiucal-tools
```

## services

setup your `/etc/hostapd/hostapd.conf`
```
systemctl enable --now hostapd.service
```

if using radiucal as a proxy (make sure to bind hostapd to not 1812 for radius)
```
# to use the default supplied proxy config
ln -s /etc/radiucal/proxy.conf /etc/radiucal/radiucal.proxy.conf
systemctl enable --now radiucal@proxy.service
```

to run an accounting server via radiucal
```
# to use the default supplied accounting config
ln -s /etc/radiucal/accounting.proxy.conf /etc/radiucal/radiucal.accounting.conf
systemctl enable --now radiucal@accounting.service
```

you may view an example config for more settings: `/etc/radiucal/example.conf`

## setup/notes

if you wish to use radiucal-tools to generate certs for hostapd
```
cd /etc/hostapd/certs
./renew.sh
```
and follow the prompts

# build (dev)

clone this repository
```
make
```

run (with a socket listening to be proxied to, e.g. hostapd-server)
```
./bin/radiucal
```

[![Build Status](https://travis-ci.org/epiphyte/radiucal.png)](https://travis-ci.org/epiphyte/radiucal)

## helpers

### radiucal-utils

tools to:
* provide adminstrative management (subset of radiucal-tools)

### radiucal-tools

tools to:
* report from radiucal
* help setup hostapd
* manage radiucal/hostapd settings

### certs

please see above but a cert generation setup is installed in the etc area for hostapd

### radiucal-bootstrap

part of both utils and tools, radiucal-bootstrap is used to manage the network configuration (netconf)
* takes pythonic definitions of users and produces an `eap_users` file that hostapd can use
* outputs report information regarding the current state of the configuration
* provides ability to create users/passwords for network access
* outputs auth attempt information

## debugging

### remotely

this requires that:
* the radius server is configured to listen/accept on the given ip below
* MAC is formatted as 00:11:22:aa:bb:cc

start with installing wpa_supplicant to get eapol_test
```
pacman -S wpa_supplicant
```

setup a test config
```
vim test.conf
---
network={
        key_mgmt=WPA-EAP
        eap=PEAP
        identity="<vlan.user>"
        password="<password>"
        phase2="autheap=MSCHAPV2"
}
```

to run
```
eapol_test -a <radius_server_ip> -c test.conf -s <secret_key> -M <mac>
```
