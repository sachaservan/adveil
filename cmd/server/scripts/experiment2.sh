#!/bin/bash

# Runs the server on each config for evaluating performance of targeting with different number of hash tables 
#
# example usage: 
#        bash experiment2.sh --port 8000 --otherhost localhost --otherport 8001 ---numprocs 1 
#
# (recall db size is 2^{dbsize})
for dbsize in 13 15 16 17 18 19 20 21
do
   for numtables in 5 10 20 30
    do
        # ad size does not matter in this experiment
        #  "$@" is all parameters that are passed to the script (and do not change between experiments)
        # limit num procs to 5 to evaluate fairly across table configs 
        bash scripts/run.sh --numads $dbsize --size 1 --numfeatures 100 --numtables $numtables --bucketsize 1 --numprocs 3 "$@"
   done
done