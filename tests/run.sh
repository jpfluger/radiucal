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
