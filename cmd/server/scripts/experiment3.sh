#!/bin/bash

# Runs the shuffling and token redemption experiment between the Broker and the CoA 
#
# example usage: 
#   On the Broker terminal: 
#        scripts/experiment3.sh --port 8000 --otherhost localhost --otherport 8001 --trials 10 ---numprocs 1  --primary
# 
#   On the CoA terminal: 
#        scripts/experiment3.sh --port 8000 --otherhost localhost --otherport 8001 ---numprocs 1 
#
# (recall numreports is 2^{numreports})
for numreports in 13 15 16 17 18 19 20 21
do
    #  "$@" is all parameters that are passed to the script (and do not change between experiments)
    bash scripts/shuffle.sh --numreports $numreports --trials 10 "$@" 
done