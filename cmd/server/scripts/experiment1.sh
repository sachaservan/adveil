#!/bin/bash

# Runs the server on each config for evaluating performance of ad retrieval.
#
# example usage: 
#     bash scripts/experiment2.sh  --port 8000 --otherhost localhost --otherport 8001 --numprocs 1 
#
# Run the server on each dataset size and ad size 
# (recall db size is 2^{dbsize})
for dbsize in 13 15 16 17 18 19 20 21
do
   for adsize in 500 1000 5000
    do
      bash scripts/run.sh --numads $dbsize --size $adsize --numfeatures 0 --numtables 0 --bucketsize 0 --noanns --numprocs 20 "$@" 
   done
done