#!/bin/bash
OUT="bin/"
USRS="../users/"
AUDIT_CSV="${OUT}audit.csv"
AUDIT_CSV_SORT="${OUT}audit.sort.csv"

rm -rf $OUT
mkdir -p $OUT
cp *.py $USRS
fail=0
python ../config_compose.py --output $OUT
fail=$?
cat $AUDIT_CSV | sort > $AUDIT_CSV_SORT
mv $AUDIT_CSV_SORT $AUDIT_CSV
for f in $(echo "audit.csv manifest eap_users"); do
    diff -u $f ${OUT}$f
    if [ $? -ne 0 ]; then
        echo "$f failed diff..."
        fail=1
    fi
done
if [ $fail -ne 0 ]; then
    exit 1
fi
