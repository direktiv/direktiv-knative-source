#!/bin/bash

export KO_DOCKER_REPO=localhost:5000

dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

echo "installing $1"

kubectl delete -f $dir/$1-source.yaml

sudo k3s crictl rmi $KO_DOCKER_REPO/$1-source:v0.0.1

ko publish ./cmd/$1-source/ -B -t v0.0.1

kubectl apply -f $dir/$1-source.yaml
