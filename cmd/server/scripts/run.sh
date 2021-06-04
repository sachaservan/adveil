#!/bin/bash

usage() { echo "Usage: $0 [-numads <log number of ads>] [--size <ad size in KB>] [--port <port>] [--numfeatures <dim of feature vectors>] [--numtables <number of tables>] [--numprocs <max num processors>] [--noanns]" 1>&2; exit 1; }

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --numads)
    NUMADS="$2"
    shift # past argument
    shift # past value
    ;;
    --size)
    SIZE="$2"
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
if  [ -z "${NUMADS}" ] || [ -z "${SIZE}" ] || [ -z "${PORT}" ] || [ -z "${NUMFEATURES}" ] || [ -z "${NUMTABLES}" ]; then
    usage
fi

boolargs=()
if [ "$NOANNS" = true ]; then 
    NOANNS=true
    boolargs+=('--noanns')
else
    NOANNS=false
fi

echo 'Number of Ads:    ' $((2**${NUMADS}))
echo 'Ad size (B):      ' ${SIZE}
echo 'Num Tables:       ' ${NUMTABLES}
echo 'Num Features:     ' ${NUMFEATURES}
echo 'Build ANNS?:      ' ${!NOANNS}
echo 'DB Parallelism:   ' ${NUMPROCS}

# build the server 
go build -o ./server ./ 

# configure experiment 
NumAds=$((2**${NUMADS})) # number of ads in total 
AdSizeBytes=${SIZE}
NumFeatures=${NUMFEATURES} # feature vector dimention for each ad 
DataMin=-50 # feature vector min value 
DataMax=50 # resp. max value 
NumTables=${NUMTABLES} # number of tables for NN search 

# TODO: make these parameters to the script 
NumProjections=50 # number of projections for NN search 
ProjectionWidth=20 # NN projection width (see Datar et al. for deets)


# server configuration 
Port=${PORT}
NumProcs=${NUMPROCS}

echo 'Running server on port:' ${PORT}

./server \
    --numads ${NumAds} \
    --adsizebytes ${AdSizeBytes} \
    --numfeatures ${NumFeatures} \
    --datamin ${DataMin} \
    --datamax ${DataMax} \
    --numtables ${NumTables} \
    --numprojections ${NumProjections} \
    --projectionwidth ${ProjectionWidth} \
    --numprocs ${NumProcs} \
    --port ${Port} \
    ${boolargs[@]}


