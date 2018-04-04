#!/bin/bash
LOCAL_CONF=~/.config/epiphyte/env
RADIUCAL_HOME=/var/lib/radiucal/
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
PREV=${USERS}.prev
if [ -e $USERS ]; then
    cp $USERS $PREV
fi

radiucal-bootstrap
if [ $? -ne 0 ]; then
    echo "unable to bootstrap (radiucal-utils|tools installed?)"
    exit 1
fi

if [ $IS_LOCAL -eq 0 ]; then
    ./monitor
fi

python netconf.py --output $PWD/$BIN
if [ $? -ne 0 ]; then
    echo "composition errors"
    exit 1
fi
diffed=1
if [ -e $USERS ]; then
    if [ -e $PREV ]; then
        changes=$(diff -u $PREV $USERS)
        diffed=$?
        if [ $diffed -ne 0 ]; then
            if [ $IS_LOCAL -eq 1 ]; then
                echo "changes" | grep -v '"pass":'
                echo
                echo "===INFO==="
                echo "the above summarizes the network changes you are making"
                echo "===INFO==="
                echo
            fi
        fi
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
        exit 0
        # run local reports
        if [ -e "./reports" ]; then
            ./local-reports
        fi
        ./reports
    fi
fi
