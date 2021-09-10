# Running the AdVeil server (Broker)

Note: due to the C++ binding for SealPIR, run the server executable using ```adveil/scripts/run_server.sh```.
The executable (```cmd/server/server```) expects ```../C/libsealwrapper.a``` to be the SealPIR library. 

### Running the Broker server
```
bash scripts/run_server.sh 
    --numads 15 \
    --size 32000 \
    --port 8000 \
    --numfeatures 100 \
    --numtables 1 \
    --bucketsize 1 \
    --numprocs 10 
```

## Running the AdVeil client
The client is used to run different experiments. 
Currently, the client is configured to:
1) query the ANNS data structure held by the server
2) retrieve an ad from the server (either via PIR or directly with no privacy)

First, configure ```run_client.sh``` with the Broker's IP address. 

Then run the client:
```
bash scripts/run_client.sh
```


## Reproducing the targeting experiments 
```
bash targeting_params.sh 
    --port 8001 \
    --numprocs 1
```
 
The client triggers the start of the experiment on the server.
The experiment script iterates through a parameters and initializes the server with the params. 
Each client run starts a new experiment under the specified parameters and saves it to a JSON file. 
Therefore, to cycle through all experiments, we re-run the client many times (e.g., 100). 
```
bash scripts/run_client.sh
```
or cycle through all experiments at once:
```
for run in {1..100}; do bash run_client.sh; sleep 60; done
```

All results as saved as ```.json``` files (one per experiment) in the ```results`` directory.
Use ```concat.py``` to merge multiple files into one ```.json``` array file. 
