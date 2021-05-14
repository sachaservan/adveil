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
cmake .
make 
```

3) Run the desired experiment!  


## Issues with running on MacOS
While the code has been tested on MacOS (Big Sur), there is a bug in the cgo interface. 
On Linux, ```uint64_t``` is cast as ```C.ulong``` in the Go code which *does not work on Mac* (seems to be a bug?)
The compiler will handle switching between two instances of ```sealpir.go``` to work around this issue. 


## Notes
- Part of the anonymous token code was obtained from the [Privacy Pass implementation](https://github.com/privacypass/challenge-bypass-server). 

## Important Warning
This implementation of AdVeil is intended as a proof-of-concept prototype only! The code was implemented for research purposes and has not been vetted by security experts. As such, no portion of the code should be used in any real-world or production setting!

