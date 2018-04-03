#!/bin/bash
OUT="actual.txt"
USRS="../users/"
AUDIT_CSV="actual.csv"
AUDIT_CSV_SORT="actual.sort.csv"

cp *.py $USRS
python ../config_compose.py --output $OUT --audit $AUDIT_CSV
diff expected.json $OUT
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
