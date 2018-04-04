BIN=bin/
AUDITS=${BIN}audit.csv
if [ ! -e $AUDITS ]; then
    exit 0
fi

MEMBERSHIP=${BIN}membership.md
echo "| vlan | user |
| ---  | --- |" > $MEMBERSHIP

cat $AUDITS | sed "s/,/ /g" | awk '{print "| " toupper($2), "|", $1 " |"}' | sort -u >> $MEMBERSHIP
