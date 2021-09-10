#!/bin/bash
for run in {1..100}; 
    do 
    bash run_client.sh "$@"; 
    
    sleep 60; 
done
