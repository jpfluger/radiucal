#!/bin/bash
RADIUCAL_HOME=/var/lib/radiucal/
IS_DAILY=/tmp/
source /etc/environment
IS_LOCAL=0
VERS="master"
echo "radiucal tools/utils: $VERS"
if [ -e "/usr/share/radiucal/radiucal-utils" ]; then
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

_sig() {
    echo "signal applications"
    kill -HUP $(pidof hostapd)
    kill -2 $(pidof radiucal)
}

cat users/user_* | sha256sum | cut -d " " -f 1 > $HASH 
if [ $IS_LOCAL -eq 0 ]; then
    daily=${IS_DAILY}.radius-$(date +%Y-%m-%d)
    if [ ! -e $daily ]; then
        _sig
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
        if [ $diffed -ne 0 ] && [ $IS_LOCAL -eq 1 ]; then
            first=1
            for f in $(echo "eap_users manifest audit.csv"); do
                fname=${BIN}$f
                p=$fname.prev
                if [ -e $fname ]; then
                    if [ -e $p ]; then
                        if [ $first -eq 1 ]; then
                            echo
                            echo "showing diff"
                            echo "============"
                            first=0
                        fi
                        diff -u $p $fname
                        echo
                    fi
                    cp $fname $p
                fi
            done
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
        _sig
        # run local reports
        if [ -e "./reports" ]; then
            ./local-reports $IS_LOCAL
        fi
        ./reports 0
    fi
fi
