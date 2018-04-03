#!/bin/bash
OUT="bin/"
USRS="../users/"
AUDIT_CSV="${OUT}audit.csv"
AUDIT_CSV_SORT="${OUT}audit.sort.csv"

rm -rf $OUT
mkdir -p $OUT
cp *.py $USRS
python ../config_compose.py --output $OUT
diff eap_users $OUT/eap_users
if [ $? -ne 0 ]; then
    echo "different composed results..."
    exit -1
fi

cat $AUDIT_CSV | sort > $AUDIT_CSV_SORT
diff audit_exp.csv $AUDIT_CSV_SORT
if [ $? -ne 0 ]; then
    echo "different audit results..."
    exit -1
fi
