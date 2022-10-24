#!/bin/bash

docker run --rm \
    -v $PWD:/workdir \
    -u $(id -u):$(id -g) \
    elama/protoc:latest generate proto
