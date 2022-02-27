cmake_minimum_required(VERSION 3.10)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

project(OnionPIR VERSION 1.0 LANGUAGES CXX)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/bin)

add_executable(main 
	main.cpp
)

add_library(sealpir STATIC
  Onion-PIR/pir.cpp
  Onion-PIR/pir_client.cpp
  Onion-PIR/pir_server.cpp
  wrapper.cpp
)

find_package(SEAL 3.5.0 EXACT REQUIRED)

target_link_libraries(main onionpir SEAL::seal -lm)

include_directories("/usr/local/include")

project(OnionPIRWrapper C)

add_library(onionwrapper STATIC
  Onion-PIR/pir.cpp
  Onion-PIR/pir_client.cpp
  Onion-PIR/pir_server.cpp
  wrapper.cpp
)

add_executable(mainc 
	main.c
)

target_link_libraries(mainc onionwrapper -lm SEAL::seal -lm)
