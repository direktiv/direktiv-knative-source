#!/bin/bash

dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

kubectl delete -f $dir/source.yaml

sudo k3s crictl rmi localhost:5000/direktiv-knative-source:v0.0.1

ko publish ./cmd/direktiv-knative-source/ -B -t v0.0.1

kubectl apply -f $dir/source.yaml
