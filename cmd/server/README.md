# Running the AdVeil server (Broker)

Note: due to the C++ binding for SealPIR, the server executable from the ```adveil/cmd/server``` directory.
Specifically, the executable expects ```../../C/libsealwrapper.a``` to be the SealPIR library. 

### Running the Broker server
```
cd adveil/cmd/server/
bash scripts/run.sh 
    --numads 15 \
    --size 32000 \
    --port 8000 \
    --numfeatures 100 \
    --numtables 1 \
    --bucketsize 1 \
    --numprocs 10 
```

## Reproducing the experiments 

### Targeting and Delivery server benchmarks

On the Broker machine we can either run the delivery or the targeting experiments. 

Delivery experiment:
```
cd adveil/cmd/server/
bash scripts/experiment1.sh 
    --port 8001 \
    --numprocs 10
```


Targeting experiment:
```
cd adveil/cmd/server/
bash scripts/experiment2.sh 
    --port 8001 \
    --numprocs 10
```
 
In both cases, the client triggers the start of the experiment on the server.
Each experiment script iterates through a parameters and initializes the server with the params. 
Each client run starts a new experiment under the specified parameters and saves it to a JSON file. 
Therefore, to cycle through all experiments, we re-run the client many times (e.g., 100). 
```
cd adveil/cmd/client
bash scripts/run.sh
```
or cycle through all experiments at once:
```
for run in {1..100}; do bash run.sh; sleep 60; done
```

This saves a ```.json``` file in the ```adveil/cmd/client``` directory (one per experiment).
Use ```adveil/cmd/concat_exp.py``` to merge multiple files into one ```.json``` array file. 



### Metrics recovery

On the Broker machine:
```
cd adveil/cmd/server/
bash scripts/experiment3.sh \ 
    --port 8000 \
    --numprocs 10 \
    --primary \
```
