#!/bin/bash
for test in command; do
    go test -cover "./${test}"
done
