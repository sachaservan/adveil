<img src="http://adveil.com/img/icon.png" alt="icon" width="100"/>

# AdVeil Prototype

Implementation for [AdVeil paper](http://adveil.com).

## Dependencies 
* [Microsoft SEAL 3.2.0](https://github.com/microsoft/SEAL/releases/tag/v3.2.0)
* [GMP Library](https://gmplib.org/) 
* CMake 
* Go 1.13 or higher 


## Getting everything to run (tested on Ubuntu 20.04.2 LTS)

0) Install dependencies:
```
sudo apt-get install build-essential
sudo apt-get install cmake
sudo apt-get install libgmp3-dev
sudo apt-get install golang-go
```

1) Globally install Microsoft SEAL 3.2.0: 

```
wget https://github.com/microsoft/SEAL/archive/refs/tags/v3.2.0.tar.gz
tar -xvf v3.2.0.tar.gz
cd SEAL-3.2.0/native/src
cmake .
make 
sudo make install
```

2) Compile the SealPIR library in ```adveil/C/```:
```
cd C/ 
cmake .
make -j 
```

3) Run the desired experiment! (See next section)


## Running experiments 

1) On the Broker machine: 
```
bash targeting_params.sh 
    --port 8000 \
    --numprocs 1
```
 
2) On the client machine:
```
bash run_client.sh --brokerhost localhost --brokerport 8000 --trials 10 --autoclose
```
or cycle through all experiments at once:
```
bash clicycle.sh --brokerhost localhost --brokerport 8000 --trials 10 --autoclose
```

The client triggers the start of the experiment on the server.
The ```targeting_params.sh``` script iterates through a parameters and initializes the server with the params. 
Each client run starts a new experiment under the specified parameters and saves it to a JSON file. 
Therefore, to cycle through all experiments, we re-run the client many times (e.g., 100); this is done automatically using ```clicycle.sh```. 
Resulting experiment summary will be saved in the ```adveil/results``` directory. 

### Converting the generated files 
All results as saved as ```.json``` files (one per experiment) in the ```adveil/results`` directory.
Use ```concat.py``` to merge multiple files into one ```.json``` array file. 
```
python concat.py --dir ../results --out ../results/targeting.json
``` 


## Issues with running on MacOS
While the code has been tested on MacOS (Big Sur), there is a bug in the cgo interface. 
On Linux, ```uint64_t``` is cast as ```C.ulong``` in the Go code which *does not work on Mac* (seems to be a bug?)
The compiler will handle switching between two instances of ```sealpir.go``` to work around this issue. 


## Notes
- Part of the anonymous token code was obtained from the [Privacy Pass implementation](https://github.com/privacypass/challenge-bypass-server). 

## Important Warning
This implementation of AdVeil is intended as a proof-of-concept prototype only! The code was implemented for research purposes and has not been vetted by security experts. As such, no portion of the code should be used in any real-world or production setting!

## License
Copyright © 2021 Sacha Servan-Schreiber

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.