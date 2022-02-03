#!/bin/bash

bytes=`cat $PWD/testdata/out.bin`

echo -n "$bytes"
echo -n "--END--"
sleep 1

echo -n "$bytes"
echo -n "--END--"
sleep 1

echo -n "$bytes"
echo -n "--END--"
sleep 1

echo -n "$bytes"
