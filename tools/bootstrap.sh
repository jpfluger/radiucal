#!/bin/bash
if [ ! -d "$PWD/users" ]; then
    echo "this location does not contain a user definition"
    exit 1
fi
DIR=/usr/share/radiucal/
for s in $(echo "configure reports"); do
    rm -f $s
    p=$DIR/$s.sh
    if [ -e $p ]; then
        cp $DIR/$s.sh $s
        chmod u+x $s
    fi
done
for f in $(echo "netconf accounts users/__config__ users/__init__"); do
    rm -f $f
    cp $DIR/$f.py $PWD/$f.py
done
