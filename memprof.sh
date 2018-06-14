#!/bin/bash
# set -e -x

# default benchmark is '.' (all)
BENCHMARK="${1:-.}"

# # comile test binary, generate memory profile and print benchmark
# go test \
#   -o decoder.test \
#   -memprofile mem.prof \
#   -memprofilerate 1 \
#   -bench ${BENCHMARK} \
#   -benchmem

# # print top memory allocators
# go tool pprof -top -alloc_objects decoder.test mem.prof

# # generate memory allocation traces
# GODEBUG=allocfreetrace=1 ./decoder.test \
#   -test.bench=${BENCHMARK} \
#   2>mem.trace

# -gcflags '-m -m'

go test -c -o decoder.test

GODEBUG=allocfreetrace=1 ./decoder.test \
  -test.run ^$ \
  -test.bench=${BENCHMARK} \
  -test.benchmem \
  2>mem.trace

# print top traces that traverse decoder.go
echo "Top allocations:"
grep 'decoder.go' mem.trace | cut -d ' ' -f 1 | sort | uniq -c | sort -r
