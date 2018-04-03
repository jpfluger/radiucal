#!/bin/bash
OUT="actual.log"
USRS="../utils/users/"
ACTUAL_KEYS="actual.keys"
KEY_LOG="actual_keys.log"
valid_mac="001122334455"
AUDIT_CSV="actual.csv"
AUDIT_CSV_SORT="actual.sort.csv"

function test-objs()
{
    echo "$1 - $2"
    echo "==="
    test-config $1 $2 "network.json"
}


function test-config()
{
    test-config-full $1 $2 $3 "keyfile.test"
}

function test-config-full()
{
    python ../utils/harness.py authorize User-Name=$1 Calling-Station-Id=$2 --json $3 --keyfile $4
}

function test-all()
{
    test-objs vlan2.user6 "000011112222"
    test-objs vlan1.user4 $valid_mac
    test-objs vlan2.user1 $valid_mac
    test-objs vlan2.user2 $valid_mac
    test-objs vlan2.user3 $valid_mac
    test-objs vlan2.user6 $valid_mac
    test-objs "AABBCCDDEE11" "aabbccddee11"
    test-objs vlan2.usera $valid_mac
}

test-all > $OUT
diff expected.log $OUT
if [ $? -ne 0 ]; then
    echo "different freepydius results..."
    exit -1
fi

for f in $(echo "b c u v"); do
    rm -f ${USRS}$f*
done
cp *.py $USRS
OUT_JSON="actual.json"
python ../utils/config_compose.py --output $OUT_JSON --audit $AUDIT_CSV
diff expected.json $OUT_JSON
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

test-config-full "dev.pwd" $valid_mac "expected.json" "keyfile.pad" > $KEY_LOG
diff expect_key.log $KEY_LOG
if [ $? -ne 0 ]; then
    echo "key decrypt failed"
    exit 1
fi

function keying-check()
{
    python ../utils/keying.py --newkey $1:abcdef --password $2 >> $ACTUAL_KEYS 2>&1
}

rm -f $ACTUAL_KEYS
keying-check 5 12
keying-check 2 1
keying-check 2 12
keying-check 0 12
python ../utils/keying.py --oldkey 4:abcdef  --newkey 2:abcdef --password "80311914048020.20111203538740" >> $ACTUAL_KEYS
sed -i '/^ / d' $ACTUAL_KEYS
sed -i "s/[0-9]3119140480[0-9]/valid/g;s/[0-9]1112035387[0-9]/valid/g" $ACTUAL_KEYS
diff expected.keys $ACTUAL_KEYS
if [ $? -ne 0 ]; then
    echo "different keying results..."
    exit -1
fi
