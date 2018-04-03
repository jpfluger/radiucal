radiucal
===

Using a go proxy+hostapd as an 802.1x RADIUS server for network authentication (or how to live without freeradius)

[![Build Status](https://travis-ci.org/epiphyte/radiucal.png)](https://travis-ci.org/epiphyte/radiucal)

# hostapd

## notes

Information that may be useful when exploring hostapd

### accept attributes

the doc mentions it but the only examples easy to find were in the hostapd tests, this is the EAP user file syntax for special attributes
```
* PEAP
"user" MSCHAPV2 "password1" [2]
radius_accept_attr=64:d:13

"user2" MSCHAPV2 "password1" [2]
radius_accept_attr=64:d:14
```

### MAB

To perform mac-bypass
```
"001122334455" MACACL "001122334455"
```
