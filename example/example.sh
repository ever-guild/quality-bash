#!/usr/bin/env bash
if [[ ${1:-} == unused ]]; then
    echo uncovered
    exit 1
fi
printf 'hello\n'
exit 0
