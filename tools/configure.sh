#!/bin/bash
LOCAL_CONF=~/.config/epiphyte/env
RADIUCAL_HOME=/var/lib/radiucal/
IS_DAILY=/tmp/
source /etc/environment
IS_LOCAL=0
if [ -e $LOCAL_CONF ]; then
    source $LOCAL_CONF
    IS_LOCAL=1
fi
echo "updating network configuration"
if [ $IS_LOCAL -eq 0 ]; then
    git reset --hard
fi
git pull

for c in $(echo "users/"); do
    cache="${c}__pycache__"
    if [ -d $cache ]; then
        echo "clearing cache: $cache"
        rm -rf $cache
    fi
done

BIN=bin/
mkdir -p $BIN
USERS=${BIN}eap_users
HASH=${BIN}last
PREV=${HASH}.prev
if [ -e $HASH ]; then
    cp $HASH $PREV
fi

cat users/user_* | sha256sum | cut -d " " -f 1 > $HASH 
if [ $IS_LOCAL -eq 0 ]; then
    ./monitor
    daily=${IS_DAILY}.radius-$(date +%Y-%m-%d)
    if [ ! -e $daily ]; then
        ./reports
        touch $daily
    fi
fi

python netconf.py --output $PWD/$BIN
if [ $? -ne 0 ]; then
    echo "composition errors"
    exit 1
fi
diffed=1
if [ -e $HASH ]; then
    if [ -e $PREV ]; then
        diff -u $PREV $HASH > /dev/null
        diffed=$?
    fi
fi

_update_files() {
    local p bname manifest
    p=${RADIUCAL_HOME}users/
    manifest=$BIN/manifest
    if [ ! -e $manifest ]; then
        echo "missing required manifest!"
        exit 1
    fi
    for e in $(find $p -type f); do
        bname=$(basename $e)
        cat $manifest | grep -q "$bname"
        if [ $? -ne 0 ]; then
            echo "dropping $bname"
            rm -f $e
        fi
    done
    for u in $(cat $manifest); do
        touch ${p}$u
    done
}

if [ $diffed -ne 0 ]; then
    echo "network configuration updated"
    if [ $IS_LOCAL -eq 0 ]; then
        git log --pretty=oneline --abbrev-commit -n 1 | smirc
        _update_files
        cp $USERS $RADIUCAL_HOME/eap_users
        echo "sighup hostapd"
        kill -HUP $(pidof hostapd)
        kill -2 $(pidof radiucal)
        # run local reports
        if [ -e "./reports" ]; then
            ./local-reports $IS_LOCAL
        fi
        ./reports 0
    fi
fi
