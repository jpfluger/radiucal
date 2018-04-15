#!/bin/bash
bin/radiucal --config tests/test.conf &
sleep 1
bin/harness --endpoint=true &
sleep 1
bin/harness
kill -2 $(pidof radiucal)
bin/harness
pkill radiucal
pkill harness
cat tests/log/radiucal.audit* | cut -d " " -f 2- > bin/results.log
diff -u bin/results.log tests/expected.log
if [ $? -ne 0 ]; then
    echo "integration test failed"
    exit 1
fi
