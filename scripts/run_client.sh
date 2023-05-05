#!/bin/bash

usage() { echo "Usage: $0 [--brokerhost <broker server addr>] [--brokerport <broker server port>] [--trials <num trials>] [--autoclose]" 1>&2; exit 1; }

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
