#!/bin/bash
# monitor auths and send to smirc
LIB=/var/lib/radiucal/
LAST=${LIB}last_audits
LAST_TIME=""
DATE=$(date +%Y-%m-%d)
SRT_LINE=0
if [ -e $LAST ]; then
    SRT_LINE=$(cat $LAST)
fi
LOG=${LIB}log/radiucal.audit.$DATE
if [ -e $LOG ]; then
    lines=$(cat $LOG | wc -l)
    cat $LOG | tail -n +$((SRT_LINE+1)) | cut -d " " -f 2- | uniq | smirc
    echo "$lines" > $LAST
fi
