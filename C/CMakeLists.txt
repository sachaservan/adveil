cmake_minimum_required(VERSION 3.10)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

project(SealPIR VERSION 2.1 LANGUAGES CXX)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/bin)

add_executable(main 
	main.cpp
)

add_library(sealpir STATIC
  SealPIR/pir.cpp
  SealPIR/pir_client.cpp
  SealPIR/pir_server.cpp
  wrapper.cpp
)

find_package(SEAL 3.2.0 EXACT REQUIRED)

target_link_libraries(main sealpir SEAL::seal -lm)

include_directories("/usr/local/include")

project(SealPIRWrapper C)

add_library(sealwrapper STATIC
  SealPIR/pir.cpp
  SealPIR/pir_client.cpp
  SealPIR/pir_server.cpp
  wrapper.cpp
)

add_executable(mainc 
	main.c
)

target_link_libraries(mainc sealwrapper -lm SEAL::seal -lm)
