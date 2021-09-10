#!/bin/bash

# Runs the server on each config for evaluating performance of ad retrieval through PIR.
#
# example usage: 
#     bash experiment1.sh  --port 8000 --numprocs 1 
#
# Run the server on each dataset size and ad size 
# Note: actual db size is 2^{dbsize}
for dbsize in 13 15 16 17 18 19 20 21
do
   for adsize in 500 1000 5000 # bytes
    do
      # 0 for most parameters because --noanns flag is set
      # "$@" contains all parameters that are passed to the script (and do not change between experiments)
      bash run_server.sh --numads $dbsize --size $adsize --numfeatures 0 --numtables 0 --bucketsize 0 --noanns --numprocs 5 "$@" 
   done
done