#!/bin/bash
if [ ! -d "$PWD/users" ]; then
    echo "this location does not contain a user definition"
    exit 1
fi
DIR=/usr/share/radiucal/
for f in $(echo "config_compose users/__config__ users/__init__"); do
    cp $DIR/$f.py $PWD/$f.py
done
