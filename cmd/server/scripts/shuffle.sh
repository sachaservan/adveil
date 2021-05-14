#!/bin/bash

usage() { echo "Usage: $0 [--numreports <log number of reports to shuffle>] [--primary] [--trials <num trials>] [--port <port>] [--otherhost <other server addr>] [--otherport <other server port>]" 1>&2; exit 1; }

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --numreports)
    NUMREPORTS="$2"
    shift # past argument
    shift # past value
    ;;
    --port)
    PORT="$2"
    shift # past argument
    shift # past value
    ;;
    --trials)
    TRIALS="$2"
    shift # past argument
    shift # past value
    ;;
    --otherhost)
    OTHERHOST="$2"
    shift # past argument
    shift # past value
    ;;
    --otherport)
    OTHERPORT="$2"
    shift # past argument
    ;;
    --primary)
    PRIMARY=true
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters


shift $((OPTIND-1))
if  [ -z "${NUMREPORTS}" ] || [ -z "${PORT}" ] || [ -z "${OTHERHOST}" ] || [ -z "${OTHERPORT}" ]; then
    usage
fi

Primary="--primary"
if  [ -z "${PRIMARY}" ]; then
    PRIMARY=false
    Primary=""
fi

if  [ -z "${TRIALS}" ]; then
    TRIALS=1
fi

ExperimentSaveFile="../../results/experiment${RANDOM}.json"


echo 'Number of tokens:' $((2**${NUMREPORTS}))
echo 'Is primary:      ' ${PRIMARY}
echo 'Num trials:      ' ${TRIALS}
echo 'Other server addr:' ${OTHERHOST}
echo 'Other server port:' ${OTHERPORT}
echo 'Saving result to ' ${ExperimentSaveFile}


# build the server 
go build -o ./server ./ 

# configure experiment 
NumReports=$((2**${NUMREPORTS})) # number of ads in total 
OtherServerAddr=${OTHERHOST}
OtherServerPort=${OTHERPORT}
Port=${PORT}
Trials=${TRIALS}

ExperimentNumETrials=5

echo Running server on port: ${PORT}

./server \
    --otherserveraddr ${OtherServerAddr} \
    --otherserverport ${OtherServerPort} \
    --experimentsavefile ${ExperimentSaveFile} \
    --experimentnumtrials ${ExperimentNumETrials} \
    --numreports ${NumReports} \
    --port ${Port} \
    --numtrials ${Trials} \
    ${Primary} \
    --justshuffle \