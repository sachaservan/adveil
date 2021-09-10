#!/bin/bash

# Runs the server on each config for evaluating performance of targeting with different number of hash tables 
#
# example usage: 
#     bash targeting_params.sh --port 8000 ---numprocs 1 
#
# Note: actual db size is 2^{dbsize}
for dbsize in 13 15 16 17 18 19 20 21
do
   for numtables in 5 10 20 30
    do
        # ad size does not impact this experiment
        # "$@" contains all parameters that are passed to the script (and do not change between experiments)
        # note: --numprocs should be small because it only affects the table-level parallelization 
        # all table queries are still executed in parallel with one-another (with each query parallelized by --numprocs). 
        bash run_server.sh --numads $dbsize --size 1 --numfeatures 100 --numtables $numtables --bucketsize 1 --numprocs 1 "$@"
   done
done