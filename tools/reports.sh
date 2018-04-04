BIN=bin/
AUDITS=${BIN}audit.csv
if [ ! -e $AUDITS ]; then
    exit 0
fi

source /etc/environment
if [ -z "$RPT_HOST" ]; then
    echo "missing RPT_HOST var"
    exit 1
fi

if [ -z "$RPT_TOKEN" ]; then
    echo "missing RPT_TOKEN var"
    exit 1
fi

_post() {
    for f in $(ls $BIN | grep "\.md"); do
        content=$(cat $BIN/$f | python -c "import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read()))")
        name=$(echo "$f" | cut -d "." -f 1)
        curl -s -k -X POST -d "name=$name&content=$content" "$RPT_HOST/reports/upload?session=$RPT_TOKEN"
    done
}

DAILY=1
if [ ! -z "$1" ]; then
    _post
    DAILY=$1
fi

# VLAN->User membership
MEMBERSHIP=${BIN}membership.md
echo "| vlan | user |
| ---  | --- |" > $MEMBERSHIP

cat $AUDITS | sed "s/,/ /g" | awk '{print "| " $2, "|", $1 " |"}' | sort -u >> $MEMBERSHIP

# User.VLAN macs assigned
ASSIGNED=${BIN}assigned.md
echo "| user | vlan | mac |
| --- | --- | --- |" > $ASSIGNED

cat $AUDITS | sed "s/,/ | /g;s/^/| /g;s/$/ |/g" | sort -u >> $ASSIGNED

if [ $DAILY -ne 1 ]; then
    exit 0
fi

# Auth information
AUTHS=${BIN}auths.md
echo "| user | mac | last |
| --- | --- | --- |" > $AUTHS

dates=$(date +%Y-%m-%d)
for i in $(seq 1 10); do
    dates="$dates "$(date -d "$i days ago" +%Y-%m-%d)
done
files=""
for d in $(echo "$dates"); do
	f="/var/lib/radiucal/log/radiucal.audit.$d"
	if [ -e $f ]; then
		files="$files $f"
	fi
done
if [ ! -z "$files" ]; then
    notcruft=""
    users=$(cat $files \
            | cut -d " " -f 3,4 \
            | sed "s/ /,/g" | sort -u)
    for u in $(echo "$users"); do
        for f in $(echo "$files"); do
            has=$(tac $f | sed "s/ /,/g" | grep "$u" | head -n 1)
            if [ ! -z "$has" ]; then
                day=$(basename $f | cut -d "." -f 3)
                stat=$(echo $has | cut -d "," -f 2 | sed "s/\[//g;s/\]//g")
                usr=$(echo $u | cut -d "," -f 1)
                notcruft="$notcruft|$usr"
                mac=$(echo $u | cut -d "," -f 2 | sed "s/(//g;s/)//g")
                echo "| $usr | $mac | $stat ($day) |" >> $AUTHS
                break
            fi
        done
    done
    notcruft=$(echo "$notcruft" | sed "s/^|//g")
    cat $AUDITS | sed "s/,/ /g" | awk '{print $2,".",$1}' | sed "s/ //g" | uniq | grep -v -E "($notcruft)" | sed "s/^/drop: /g" | sort -u | smirc
fi

# Leases
LEASES=${BIN}leases.md
echo "| mac | ip |
| --- | --- |" > $LEASES
unknowns=""
leases=$(curl -s -k "$RPT_HOST/reports/view/dns?raw=true")
for l in $(echo "$leases"); do
    t=$(echo $l | cut -d "," -f 1)
    if [[ "$t" == "static" ]]; then
        continue
    fi
    ip=$(echo $l | cut -d "," -f 3)
    mac=$(echo $l | cut -d "," -f 2 | tr '[:upper:]' '[:lower:]' | sed "s/://g")
    if [ ! -z "$LEASE_MGMT" ]; then
        echo "$ip" | grep -q "$LEASE_MGMT"
        if [ $? -eq 0 ]; then
            continue
        fi
    fi
    cat $AUDITS | grep -q "$mac"
    if [ $? -eq 0 ]; then
        continue
    fi
    unknowns="$unknowns $mac ($ip)"
    echo "| $mac | $ip |" >> $LEASES
done
if [ -z "$unknowns" ]; then
    echo "| n/a | n/a |" >> $LEASES
else
    echo "unknown leases: $unknowns" | smirc
fi

_post
