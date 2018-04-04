BIN=bin/
AUDITS=${BIN}audit.csv
if [ ! -e $AUDITS ]; then
    exit 0
fi

MEMBERSHIP=${BIN}membership.md
echo "| vlan | user |
| ---  | --- |" > $MEMBERSHIP

cat $AUDITS | sed "s/,/ /g" | awk '{print "| " $2, "|", $1 " |"}' | sort -u >> $MEMBERSHIP

ASSIGNED=${BIN}assigned.md
echo "| user | vlan | mac |
| --- | --- | --- |" > $ASSIGNED

cat $AUDITS | sed "s/,/ | /g;s/^/| /g;s/$/ |/g" | sort -u >> $ASSIGNED
