#!/bin/bash

###############################################
# Contextual vs Targeted Accuracy experiment
###############################################

# build the experiemnt binary 
go build -o main

# configure experiment according to Datar, Mayur, et al. 
# "Locality-sensitive hashing scheme based on p-stable distributions.
DatasetSize=1000
NumFeatures=100
DataMin=-50
DataMax=50
NumQueries=10000
NumProjections=10
ProjectionWidth=200
ApproximationFactor=2.0
NumTables=(5 10 20 30)
NumNNPerQuery=1
MaxDistanceToNN=100

# how far a user profile can be from the website context 
MaxDistanceFromContext=(250 275 300 325 350 375 400) 

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