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
    users=$(cat $files \
            | cut -d " " -f 3,4 \
            | sed "s/ /,/g" | sort -u)
    for u in $(echo "$users"); do
        for f in $(echo "$files"); do
            has=$(cat $f | sed "s/ /,/g" | grep "$u" | head -n 1)
            if [ ! -z "$has" ]; then
                day=$(basename $f | cut -d "." -f 3)
                stat=$(echo $has | cut -d "," -f 2 | sed "s/\[//g;s/\]//g")
                usr=$(echo $u | cut -d "," -f 1)
                mac=$(echo $u | cut -d "," -f 2 | sed "s/(//g;s/)//g")
                echo "| $usr | $mac | $stat ($day) |" >> $AUTHS
                break
            fi
        done
    done
fi

