radiucal
===

Using a go proxy+hostapd as an 802.1x RADIUS server for network authentication (or how to live without freeradius)

# purpose

This is a go proxy+hostapd setup that provides a very simple configuration to manage 802.1x authentication and management on a LAN.

Expectations:
* Running on archlinux as a host/server
* hostapd can do a lot with EAP and RADIUS as a service, this should serve as an exploration of these features
* Fully replace freeradius for 802.1x/AAA/etc.

# requirements

## AAA

* Authentication (Your driver's license proves that you're the person you say you are)
* Authorization (Your driver's license lets you drive your car, motorcycle, or CDL)
* Accounting (A log shows that you've driven on these roads at a certain date)

## Goals:
* Support a port-restricted LAN (+wifi) in a controlled, physical area
* Provide a singular authentication strategy for supported clients using peap+mschapv2 (no CA validation).
* Windows 10
* Arch/Fedora Linux (any supporting modern versions of NetworkManager or systemd-networkd when server/headless)
* Android 7+
* Map authenticated user+MAC combinations to specific VLANs
* Support MAC-based authentication (bypass) for systems that can not authenticate themselves
* Integrate with Ubiquiti devices
* Avoid client-issued certificates (and management)
* Centralized configuration file
* As few open endpoints as possible on the radius server (only open ports 1812 and 1813 for radius)
* Avoid deviations from the standard/installed freeradius configurations

**These goals began with our usage of freeradius and continue to be vital to our operation**

# install

install from the epiphyte [repository](https://mirror.epiphyte.network/repos)
```
pacman -S hostapd-server radiucal radicual-tools
```

adminstrative machines should install
```
pacman -S radiucual-utils
```

## services

setup your `/etc/hostapd/hostapd.conf`
```
systemctl enable --now hostapd.service
```

if using radiucal (make sure to bind hostapd to not 1812 for radius)
```
systemctl enable --now radiucal.service
# for accounting
systemctl enable --now radiucal-accounting.service
```

## setup/notes

if you wish to use radiucal-tools to generate certs
```
cd /etc/hostapd/certs
./renew.sh
```
and follow the prompts

# components

## radiucal

radiucal is a go proxy that receives UDP packets and routes them along (namely to hostapd/another radius server)

the proxy:
* provides a preauth user+mac filter check
* logs preauth success/failure
* provides a cut-in for debugging
* overrides the concept of "radius_clients" as all will have to have a single shared secret

### build (dev)

clone this repository
```
make
```

run (with a socket listening to be proxied to, e.g. hostapd-server)
```
./bin/radiucal
```

[![Build Status](https://travis-ci.org/epiphyte/radiucal.png)](https://travis-ci.org/epiphyte/radiucal)

## hostapd

Information that may be useful when exploring hostapd and/or manually configuring an 802.1x server. The `hostapd-server` PKGBUILD is available in [here](https://github.com/epiphyte/pkgbuilds) to see what build settings are required

### interface

You do _not_ have to bind an interface when using `hostapd-server` packaging

### debugging

this is documented but to to see debug output
```
hostapd -dd /etc/hostapd/hostapd.conf
```

### eap users

information about the eap user definitions

#### accept attributes

the doc mentions it but the only examples easy to find were in the hostapd tests, this is the EAP user file syntax for special attributes
```
# PEAP indicates phase1 for all clients
* PEAP

# allow this user with an attribute
"user" MSCHAPV2 "password1" [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:d:2

# and this one with another
"user2" MSCHAPV2 "password1" [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:d:1
```

#### MAB

To perform mac-bypass for a mac
```
"001122334455" MD5 "001122334455"
```

## radiucal-utils

tools to:
* provide adminstrative management (subset of radiucal-tools)

## radiucal-tools

tools to:
* report from radiucal
* help setup hostapd
* manage radiucal/hostapd settings

### certs

please see above but a cert generation setup is installed in the etc area for hostapd

## radiucal-bootstrap

part of both utils and tools, radiucal-bootstrap is used to manage the network configuration (netconf)
* takes pythonic definitions of users and produces an `eap_users` file that hostapd can use
* outputs report information regarding the current state of the configuration
* provides ability to create users/passwords for network access
* outputs auth attempt information

## connecting

### headless/server

to connect a headless system using systemd-network and wpa_supplicant check [here](https://github.com/epiphyte/wsw)

### android

* EAP method: PEAP
* Phase 2: MSCHAPV2
* CA Cert: Do not validate
* Identity: <vlan.user>
* Password: <pass>

### network-manager (nm-applet)

Create a connection and go to the "802.1X Security" tab

* Check the "Use 802.1X" box
* Auth: Protected EAP (PEAP)
* Check the "No CA certificate is required"
* PEAP version: Automatic
* Inner authentication: MSCHAPv2
* Username: <vlan.user>
* Password: <password>


## debugging

### remotely

this requires that:
* the radius server is configured to listen/accept on the given ip below (e.g. iptables and client.conf setup)
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
