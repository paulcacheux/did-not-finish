#!/bin/bash

docker build . -t dnf
docker run -it dnf