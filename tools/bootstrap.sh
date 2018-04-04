#!/bin/bash
if [ ! -d "$PWD/users" ]; then
    echo "this location does not contain a user definition"
    exit 1
fi
DIR=/usr/share/radiucal/
for s in $(echo "configure monitor reports"); do
    cp $DIR/$s.sh $s
    chmod u+x $s
done
for f in $(echo "netconf accounts users/__config__ users/__init__"); do
    cp $DIR/$f.py $PWD/$f.py
done
