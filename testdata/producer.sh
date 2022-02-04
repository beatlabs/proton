#!/bin/bash

arg=$1

bytes=`cat $PWD/testdata/out.bin`

echo -n "$bytes"
echo -n "${arg}"
sleep 1

echo -n "$bytes"
echo -n "${arg}"
sleep 1

echo -n "$bytes"
echo -n "${arg}"
sleep 1

echo -n "$bytes"
