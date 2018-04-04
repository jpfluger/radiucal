#!/bin/bash
if [ ! -d "$PWD/users" ]; then
    echo "this location does not contain a user definition"
    exit 1
fi
DIR=/usr/share/radiucal/
cp $DIR/configure.sh configure
chmod u+x configure
for f in $(echo "config_compose new_user users/__config__ users/__init__"); do
    cp $DIR/$f.py $PWD/$f.py
done
