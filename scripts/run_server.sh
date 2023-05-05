#!/bin/bash

usage() { echo "Usage: $0 [-numcat <log number of ads>] [--port <port>] [--numfeatures <dim of feature vectors>] [--numtables <number of tables>] [--numprobes <number of multiprobes>] [--numprocs <max num processors>]" 1>&2; exit 1; }

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --numcat)
    NUMCAT="$2"
    shift # past argument
    shift # past value
    ;;
    --numfeatures)
    NUMFEATURES="$2"
    shift # past argument
    shift # past value
    ;;
    --port)
    PORT="$2"
    shift # past argument
    shift # past value
    ;;
    --numtables)
    NUMTABLES="$2"
    shift # past argument
    shift # past value
    ;;
    --numprobes)
    NUMPROBES="$2"
    shift # past argument
    shift # past value
    ;;
    --noanns)
    NOANNS=true
    shift # past argument
    ;;
    --numprocs)
    NUMPROCS="$2"
    shift # past argument
    shift # past value
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters


shift $((OPTIND-1))
if  [ -z "${NUMCAT}" ] || [ -z "${PORT}" ] || [ -z "${NUMFEATURES}" ] || [ -z "${NUMTABLES}" ] || [ -z "${NUMPROBES}" ]; then
    usage
fi


echo 'Number of categories:    ' $((2**${NUMCAT}))
echo 'Num Tables:       ' ${NUMTABLES}
echo 'Num Probes:       ' ${NUMPROBES}
echo 'Num Features:     ' ${NUMFEATURES}
echo 'DB Parallelism:   ' ${NUMPROCS}

# build the server 
go build -o ../cmd/server ../cmd/server/

# configure experiment 
NumCategories=$((2**${NUMCAT})) # number of categories in total 
NumFeatures=${NUMFEATURES} # feature vector dimention for each ad 
DataMin=-50 # feature vector min value 
DataMax=50 # resp. max value 
NumTables=${NUMTABLES} # number of tables for NN search 
NumProbes=${NUMPROBES} # number of probes per hash table 

# TODO: make these parameters to the script 
NumProjections=50 # number of projections for NN search 
ProjectionWidth=20 # NN projection width (see Datar et al. for deets)


# server configuration 
Port=${PORT}
NumProcs=${NUMPROCS}

echo 'Running server on port:' ${PORT}

../cmd/server/server \
    --numcategories ${NumCategories} \
    --numfeatures ${NumFeatures} \
    --datamin ${DataMin} \
    --datamax ${DataMax} \
    --numtables ${NumTables} \
    --numprobes ${NumProbes} \
    --numprojections ${NumProjections} \
    --projectionwidth ${ProjectionWidth} \
    --numprocs ${NumProcs} \
    --port ${Port}

