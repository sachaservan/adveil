#!/bin/bash

###############################################
# Contextual vs Targeted Accuracy experiment
###############################################

# build the experiemnt binary 
go build -o main

# configure experiment according to Datar, Mayur, et al. 
# "Locality-sensitive hashing scheme based on p-stable distributions.
DatasetSize=10000 # this only impacts the number of false-positives with high probability 
NumFeatures=100
DataMin=-255
DataMax=255
NumQueries=100000
NumProjections=10
ProjectionWidth=500
ApproximationFactor=2.0
NumTables=(5 10 20 30)
NumNNPerQuery=1
MaxDistanceToNN=100

# how far a user profile can be from the website context 
MaxDistanceFromContext=(50 75 100 150 200 250) 

SaveFileName="exp_contextual_vs_targeted.json"

# run the experiemnts with the specified parameters 
./main \
    --datasetsize ${DatasetSize} \
    --numfeatures ${NumFeatures} \
    --numqueries ${NumQueries} \
    --projectionwidth ${ProjectionWidth} \
    --approximationfactor ${ApproximationFactor} \
    --numprojections ${NumProjections} \
    --datamin ${DataMin} \
    --datamax ${DataMax} \
    --numtables ${NumTables[@]} \
    --numnn ${NumNNPerQuery} \
    --maxdistancetonn ${MaxDistanceToNN} \
    --maxdistancefromcontext ${MaxDistanceFromContext[@]} \
    --savefilename ${SaveFileName}