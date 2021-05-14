## Running the AdVeil client
The client is used to run different experiments. 
Currently, the client is configured to:
1) query the ANNS data structure held by the server
2) retrieve an ad from the server (either via PIR or directly with no privacy)

First, configure ```adveil/cmd/client/scripts/run.sh``` with the Broker and the CoA's IP addresses. 

Then run the client:
```
cd adveil/cmd/client
bash scripts/run.sh
```





