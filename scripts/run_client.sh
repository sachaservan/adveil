#!/bin/bash

# Command line arguments to run the client: 
# ServerAddrs: array of server IP addresses; if only one addr provided, assumes single server setting 
# ServerPorts: array of server ports 
# AutoCloseClient: if YES then kills the client once all requests havve completed 
# EvaluatePrivateANN: if YES then performs a private ANN retrieval of nearest neighbors to the clients profile 
# EvaluateAdRetrieval: if YES then does a PIR query to retrieve an ad from the server 
# ExperimentNumETrials: number of times to run each experiment 

usage() { echo "Usage: $0 [--brokerhost <broker server addr>] [--brokerport <broker server port>] [--trials <num trials>] [--targeting] [--delivery] [--autoclose]" 1>&2; exit 1; }

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --brokerhost)
    HOST="$2"
    shift # past argument
    shift # past value
    ;;
    --brokerport)
    PORT="$2"
    shift # past argument
    shift # past value
    ;;
    --trials)
    TRIALS="$2"
    shift # past argument
    shift # past value
    ;;
    --brokerport)
    PORT="$2"
    shift # past argument
    shift # past value
    ;;
    --autoclose)
    AUTOCLOSE=true
    shift # past argument
    ;;
    --targeting)
    TARGETING=true
    shift # past argument
    ;;
    --delivery)
    DELIVERY=true
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

shift $((OPTIND-1))
if  [ -z "${TRIALS}" ] || [ -z "${PORT}" ] || [ -z "${HOST}" ]; then
    usage
fi

# build the client 
go build -o ../cmd/client/ ../cmd/client/ 

# add the boolean flags
boolargs=()
if [ "$TARGETING" = true ]; then 
    boolargs+=('--evaluateprivateann')
fi 
if [ "$DELIVERY" = true ]; then
    boolargs+=('--evaluateadretrieval')
fi 
if [ "$AUTOCLOSE" = true ]; then 
    boolargs+=('--autocloseclient')
fi 

# configure arguments 
BrokerHost=${HOST}
BrokerPort=${PORT}
ExperimentNumTrials=${TRIALS}
ExperimentSaveFile="../results/experiment${RANDOM}${RANDOM}.json"

echo 'Broker IP addr: ' ${HOST}':'${PORT}
echo 'Num trials:     ' ${ExperimentNumTrials}
echo 'Flags:          ' ${boolargs[@]}
echo 'Saving to       ' ${ExperimentSaveFile}

# run the experiemnts with the specified parameters 
../cmd/client/client \
    --experimentsavefile ${ExperimentSaveFile} \
    --serveraddr ${BrokerHost} \
    --serverport ${BrokerPort} \
    --experimentnumtrials ${ExperimentNumTrials} \
    ${boolargs[@]}
