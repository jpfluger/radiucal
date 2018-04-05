#!/bin/bash
# monitor auths and send to smirc
LIB=/var/lib/radiucal/
LAST_TIME=""
DATE=$(date +%Y-%m-%d)
AUDITS=${LIB}proxy
LAST=$AUDITS.$DATE
SRT_LINE=0
if [ -e $LAST ]; then
    SRT_LINE=$(cat $LAST)
else
    rm -f ${AUDITS}*
fi
echo "$SRT_LINE"
LOG=${LIB}log/radiucal.audit.$DATE
if [ -e $LOG ]; then
    lines=$(cat $LOG | wc -l)
    cat $LOG | tail -n +$((SRT_LINE+1)) | cut -d " " -f 2- | sort -u | smirc --private
    echo "$lines" > $LAST
fi
